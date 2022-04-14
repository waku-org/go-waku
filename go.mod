module github.com/status-im/go-waku

go 1.15

replace github.com/raulk/go-watchdog v1.2.0 => github.com/status-im/go-watchdog v1.2.0-ios-nolibproc

require (
	contrib.go.opencensus.io/exporter/prometheus v0.4.1
	github.com/cruxic/go-hmac-drbg v0.0.0-20170206035330-84c46983886d
	github.com/ethereum/go-ethereum v1.10.16
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/rpc v1.2.0
	github.com/ipfs/go-ds-sql v0.3.0
	github.com/ipfs/go-log v1.0.5
	github.com/libp2p/go-libp2p v0.18.0
	github.com/libp2p/go-libp2p-connmgr v0.3.1
	github.com/libp2p/go-libp2p-core v0.14.0
	github.com/libp2p/go-libp2p-peerstore v0.6.0
	github.com/libp2p/go-libp2p-pubsub v0.6.1
	github.com/libp2p/go-libp2p-quic-transport v0.16.1
	github.com/libp2p/go-msgio v0.1.0
	github.com/libp2p/go-tcp-transport v0.5.1
	github.com/libp2p/go-ws-transport v0.6.1-0.20220221074654-eeaddb3c061d
	github.com/mattn/go-sqlite3 v1.14.12
	github.com/multiformats/go-multiaddr v0.5.0
	github.com/status-im/go-discover v0.0.0-20220406135310-85a2ce36f63e
	github.com/status-im/go-waku-rendezvous v0.0.0-20211018070416-a93f3b70c432
	github.com/stretchr/testify v1.7.1
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/urfave/cli/v2 v2.4.0
	go.opencensus.io v0.23.0
	go.uber.org/zap v1.21.0
	google.golang.org/protobuf v1.28.0 // indirect
)
