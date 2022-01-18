package main

import (
	"fmt"
	"os"

	logging "github.com/ipfs/go-log"
	"github.com/jessevdk/go-flags"
	"github.com/status-im/go-waku/waku"
	"github.com/status-im/go-waku/waku/v2/utils"
)

var options waku.Options

var parser = flags.NewParser(&options, flags.Default)

func main() {
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}

	// for go-libp2p loggers
	lvl, err := logging.LevelFromString(options.LogLevel)
	if err != nil {
		os.Exit(1)
	}
	logging.SetAllLoggers(lvl)

	// go-waku logger
	fmt.Println(options.LogLevel)
	err = utils.SetLogLevel(options.LogLevel)
	if err != nil {
		os.Exit(1)
	}

	waku.Execute(options)
}
