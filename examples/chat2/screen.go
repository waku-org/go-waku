package main

import (
	"os"

	"golang.org/x/term"
)

func GetTerminalDimensions() (int, int) {
	physicalWidth, physicalHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic("Could not determine terminal size")
	}
	return physicalWidth, physicalHeight
}
