package utils

import (
	"crypto/sha256"
	"hash"
	"sync"
)

var sha256Pool = sync.Pool{New: func() interface{} {
	return sha256.New()
}}

func SHA256(data []byte) []byte {
	h, ok := sha256Pool.Get().(hash.Hash)
	if !ok {
		h = sha256.New()
	}
	defer sha256Pool.Put(h)
	h.Reset()
	h.Write(data)
	return h.Sum(nil)
}
