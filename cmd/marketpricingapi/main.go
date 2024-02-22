package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"cloud.google.com/go/storage"

	"github.com/urfave/cli"
	"go.uber.org/zap"

	"github.com/KyberNetwork/cclog/lib/app"
)

const (
	flagBaseDir  = "base-dir"
	flagBindAddr = "bind-addr"
	flagBucketID = "bucket-id"
)

var sugar = zap.NewExample().Sugar()

func main() {
	zapp := app.NewApp()
	zapp.Name = "Log receiving api"
	zapp.Usage = "Log receiving api"
	zapp.Action = run

	zapp.Flags = append(zapp.Flags,
		cli.StringFlag{
			Name:   flagBaseDir,
			Usage:  "log file base dir",
			Value:  "/var/lib/kyber-cclog/data/prod-market-pricing",
			EnvVar: "LOG_BASE_DIR",
		},
		cli.StringFlag{
			Name:   flagBucketID,
			Usage:  "bucket id",
			Value:  "internal-cclog-8aa125ee",
			EnvVar: "BUCKET_ID",
		},
		cli.StringFlag{
			Name:   flagBindAddr,
			Usage:  "bind address",
			Value:  ":4565",
			EnvVar: "BIND_ADDR",
		},
	)

	if err := zapp.Run(os.Args); err != nil {
		sugar.Errorw("service stopped", "error", err)
		_ = sugar.Sync()
	}
}

func writeJSON(w http.ResponseWriter, code int, data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(dataBytes)
	return err
}

func run(c *cli.Context) error {
	var (
		f   func()
		err error
	)
	sugar, f, err = app.NewSugaredLogger(c)
	if err != nil {
		return err
	}
	defer f()
	zap.ReplaceGlobals(sugar.Desugar())
	l := zap.S()

	ctx := context.Background()
	// Sets your Google Cloud Platform project ID.
	// projectID := "production-021722"

	// Creates a client.
	client, err := storage.NewClient(ctx)
	if err != nil {
		l.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()
	finder := NewFinder(NewGCSClient(client),
		c.String(flagBaseDir), c.String(flagBucketID))
	http.HandleFunc("GET /orderbook/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			_ = writeJSON(w, http.StatusBadRequest, map[string]string{"err": "empty id"})
			return
		}
		data, err := finder.GetLogRecord("prod-market-pricing", id)
		if err != nil {
			_ = writeJSON(w, http.StatusBadRequest, map[string]string{"err": err.Error()})
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(data))
	})
	l.Fatal(http.ListenAndServe(c.String(flagBindAddr), nil))
	return nil
}
