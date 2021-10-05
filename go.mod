module github.com/status-im/go-waku

go 1.15

replace github.com/ethereum/go-ethereum v1.10.4 => github.com/status-im/go-ethereum v1.10.4-status.2

require (
	contrib.go.opencensus.io/exporter/prometheus v0.3.0
	github.com/cruxic/go-hmac-drbg v0.0.0-20170206035330-84c46983886d
	github.com/ethereum/go-ethereum v1.10.4
	github.com/golang/protobuf v1.5.2
	github.com/ipfs/go-ds-sql v0.2.0
	github.com/ipfs/go-log v1.0.4
	github.com/jessevdk/go-flags v1.4.0
	github.com/libp2p/go-libp2p v0.13.0
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/libp2p/go-libp2p-peerstore v0.2.6
	github.com/libp2p/go-msgio v0.0.6
	github.com/mattn/go-sqlite3 v1.14.6
	github.com/minio/sha256-simd v0.1.1
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multiaddr-net v0.2.0
	github.com/prometheus/common v0.29.0 // indirect
	github.com/prometheus/statsd_exporter v0.21.0 // indirect
	github.com/status-im/go-waku-rendezvous v0.0.0-20211005020656-b53661c58574
	github.com/status-im/go-wakurelay-pubsub v0.4.3-0.20210729162817-adc68830282a
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	go.opencensus.io v0.23.0
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e // indirect
)
