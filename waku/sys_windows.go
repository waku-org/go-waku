//go:build windows

package waku

import (
	"math"
)

func getNumFDs() int {
	return math.MaxInt
}
