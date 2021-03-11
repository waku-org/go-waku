package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/status-im/go-waku/waku/v2/node"
	//node "waku/v2/node"
)

func main() {
	golog.SetAllLoggers(golog.LevelInfo) // Change to INFO for extra info

	hostAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:5555")
	extAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:5555")

	var r io.Reader
	r = rand.Reader
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.ECDSA, -1, r)
	if err != nil {
		panic(err)
	}

	wakuNode, err := node.New(prvKey, hostAddr, extAddr)

	if err != nil {
		fmt.Print(err)
	}

	_ = wakuNode // TODO: Just to shut up the compiler. Do a proper test case and remove this

	select {} // Run forever

}
