module github.com/status-im/go-waku

go 1.15

replace github.com/ethereum/go-ethereum v1.10.4 => github.com/status-im/go-ethereum v1.10.4-status.2

require (
	contrib.go.opencensus.io/exporter/prometheus v0.4.0
	github.com/cruxic/go-hmac-drbg v0.0.0-20170206035330-84c46983886d
	github.com/ethereum/go-ethereum v1.10.16
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/rpc v1.2.0
	github.com/ipfs/go-ds-sql v0.3.0
	github.com/ipfs/go-log v1.0.5
	github.com/jessevdk/go-flags v1.5.0
	github.com/libp2p/go-libp2p v0.17.0
	github.com/libp2p/go-libp2p-connmgr v0.3.1
	github.com/libp2p/go-libp2p-core v0.13.0
	github.com/libp2p/go-libp2p-peerstore v0.6.0
	github.com/libp2p/go-libp2p-pubsub v0.6.1
	github.com/libp2p/go-msgio v0.1.0
	github.com/mattn/go-sqlite3 v1.14.11
	github.com/minio/sha256-simd v1.0.0
	github.com/multiformats/go-multiaddr v0.5.0
	github.com/status-im/go-discover v0.0.0-20211207185514-87bd8c3fe429
	github.com/status-im/go-waku-rendezvous v0.0.0-20211018070416-a93f3b70c432
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	go.opencensus.io v0.23.0
	go.uber.org/zap v1.21.0
)
