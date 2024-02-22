package main

import (
	"context"
	"os"

	"cloud.google.com/go/storage"

	"github.com/urfave/cli"
	"go.uber.org/zap"

	"github.com/KyberNetwork/cclog/lib/app"
)

const (
	flagBaseDir  = "base-dir"
	flagBindAddr = "bind-addr"
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
			Value:  "data/log/",
			EnvVar: "LOG_BASE_DIR",
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
	defer client.Close()

	id := "cnapifvdqqbb5kth3kh0" //"cna6q3fdqqbb5ksd8sr0"
	bucketName := "internal-cclog-8aa125ee"
	finder := NewFinder(NewGCSClient(client),
		"/var/lib/kyber-cclog/data/prod-market-pricing", bucketName)
	res, err := finder.GetLogRecord("prod-market-pricing", id)
	if err != nil {
		panic(err)
	}
	// fmt.Println("result:", res)
	os.WriteFile("/home/secmask/result.txt", []byte(res), 0o644)
	return nil
}
