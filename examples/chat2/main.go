package main

import (
	"os"

	logging "github.com/ipfs/go-log"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/urfave/cli/v2"
)

var options Options

func main() {
	app := &cli.App{
		Flags: getFlags(),
		Action: func(c *cli.Context) error {
			// for go-libp2p loggers
			logLevel := "panic" // to mute output from logs
			lvl, err := logging.LevelFromString(logLevel)
			if err != nil {
				return err
			}
			logging.SetAllLoggers(lvl)

			// go-waku logger
			err = utils.SetLogLevel(logLevel)
			if err != nil {
				return err
			}

			execute(options)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
