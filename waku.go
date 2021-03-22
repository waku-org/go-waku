package main

import (
	logging "github.com/ipfs/go-log"
	"github.com/status-im/go-waku/cmd"
)

func main() {
	lvl, err := logging.LevelFromString("info")
	if err != nil {
		panic(err)
	}
	logging.SetAllLoggers(lvl)

	cmd.Execute()
}
