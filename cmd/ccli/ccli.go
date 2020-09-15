package main

import (
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli"

	"github.com/KyberNetwork/cclog/lib/client"
)

const (
	flagRemoteAddr = "remote-addr"
	flagName       = "name"
)

func main() {
	app := cli.NewApp()
	app.Name = "cli to send log to server"
	app.Usage = "cli to send log to server"
	app.Action = run

	app.Flags = append(app.Flags,
		cli.StringFlag{
			Name:   flagRemoteAddr,
			Usage:  "remote address",
			Value:  "127.0.0.1:4560",
			EnvVar: "REMOTE_ADDR",
		},
		cli.StringFlag{
			Name:   flagName,
			Usage:  "name of log file",
			Value:  "test",
			EnvVar: "LOG_NAME",
		},
	)
	if err := app.Run(os.Args); err != nil {
		fmt.Println("run error", err)
	}
}

func run(c *cli.Context) error {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		fmt.Println("nothing to send")
		return nil
	}
	w2 := client.NewSyncLogClient(c.String(flagName), c.String(flagRemoteAddr))
	n, err := io.Copy(w2, os.Stdin)
	if err != nil {
		fmt.Println("write failed", err)
		return nil
	}
	fmt.Println("done with", n, "bytes")
	return nil
}
