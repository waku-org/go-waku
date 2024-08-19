package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap/zapcore"
)

var options Options

func main() {
	app := &cli.App{
		Flags: getFlags(),
		Action: func(c *cli.Context) error {

			lvl, err := zapcore.ParseLevel(options.LogLevel)
			if err != nil {
				return err
			}

			logging.SetAllLoggers(logging.LogLevel(lvl))
			utils.InitLogger("console", "file:chat2.log", "chat2", lvl)

			execute(options)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
