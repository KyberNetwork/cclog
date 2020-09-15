package main

import (
	"os"

	"github.com/urfave/cli"
	"go.uber.org/zap"

	"github.com/KyberNetwork/cclog/lib/app"
	"github.com/KyberNetwork/cclog/lib/server"
)

const (
	flagBaseDir     = "base-dir"
	flagBindAddr    = "bind-addr"
	flagMaxFileSize = "max-file-size"
)

var sugar = zap.NewExample().Sugar()

func main() {
	app := app.NewApp()
	app.Name = "Log Server"
	app.Usage = "Log Server to receive log from services"
	app.Action = run

	app.Flags = append(app.Flags,
		cli.StringFlag{
			Name:   flagBaseDir,
			Usage:  "log file base dir",
			Value:  "data/log/",
			EnvVar: "LOG_BASE_DIR",
		},
		cli.StringFlag{
			Name:   flagBindAddr,
			Usage:  "bind address",
			Value:  ":4560",
			EnvVar: "BIND_ADDR",
		},
		cli.Uint64Flag{
			Name:   flagMaxFileSize,
			Usage:  "max file size in MB",
			Value:  2048,
			EnvVar: "MAX_FILE_SIZE",
		},
	)

	if err := app.Run(os.Args); err != nil {
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
	maxSize := c.Uint64(flagMaxFileSize)
	if maxSize <= 0 {
		sugar.Fatalw("max size should > 0")
	}
	wm := server.NewWriterMan(c.String(flagBaseDir), maxSize*1024*1024)
	server := server.NewServer(c.String(flagBindAddr), wm)
	sugar.Infow("server now start", "bind_addr", c.String(flagBindAddr))
	return server.Start()
}
