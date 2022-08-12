//go:build !gowaku_rln
// +build !gowaku_rln

package main

import "github.com/urfave/cli/v2"

func rlnFlags() []cli.Flag {
	return nil
}
