module github.com/status-im/go-waku

go 1.15

replace github.com/raulk/go-watchdog v1.2.0 => github.com/status-im/go-watchdog v1.2.0-ios-nolibproc

replace github.com/ethereum/go-ethereum v1.10.17 => github.com/status-im/go-ethereum v1.10.4-status.2

require (
	contrib.go.opencensus.io/exporter/prometheus v0.4.1
	github.com/cruxic/go-hmac-drbg v0.0.0-20170206035330-84c46983886d
	github.com/ethereum/go-ethereum v1.10.17
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/golang-migrate/migrate/v4 v4.15.2
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/rpc v1.2.0
	github.com/ipfs/go-ds-sql v0.3.0
	github.com/ipfs/go-log v1.0.5
	github.com/kr/pretty v0.3.0 // indirect
	github.com/lib/pq v1.10.3 // indirect
	github.com/libp2p/go-libp2p v0.18.0
	github.com/libp2p/go-libp2p-connmgr v0.3.1
	github.com/libp2p/go-libp2p-core v0.14.0
	github.com/libp2p/go-libp2p-peerstore v0.6.0
	github.com/libp2p/go-libp2p-pubsub v0.6.1
	github.com/libp2p/go-libp2p-quic-transport v0.16.1
	github.com/libp2p/go-msgio v0.1.0
	github.com/libp2p/go-tcp-transport v0.5.1
	github.com/libp2p/go-ws-transport v0.6.1-0.20220221074654-eeaddb3c061d
	github.com/mattn/go-sqlite3 v2.0.2+incompatible
	github.com/multiformats/go-multiaddr v0.5.0
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.17.0 // indirect
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/status-im/go-discover v0.0.0-20220406135310-85a2ce36f63e
	github.com/status-im/go-waku-rendezvous v0.0.0-20211018070416-a93f3b70c432
	github.com/stretchr/testify v1.7.1
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/urfave/cli/v2 v2.4.0
	go.opencensus.io v0.23.0
	go.uber.org/zap v1.21.0
	golang.org/x/sync v0.0.0-20220513210516-0976fa681c29
	golang.org/x/tools v0.1.7 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)
