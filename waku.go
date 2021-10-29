package main

import (
	"os"

	logging "github.com/ipfs/go-log"
	"github.com/jessevdk/go-flags"
	"github.com/status-im/go-waku/waku"
)

var options waku.Options

var parser = flags.NewParser(&options, flags.Default)

func main() {
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}

	lvl := logging.LevelFromString(options.LogLevel)

	logging.SetAllLoggers(lvl)

	waku.Execute(options)
}
