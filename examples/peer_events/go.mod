module peer_events

go 1.15

// replace github.com/status-im/go-waku => ../..

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.12

require (
	github.com/ethereum/go-ethereum v1.10.4
	github.com/ipfs/go-log v1.0.5
	github.com/status-im/go-waku v0.0.0-20211012131444-baf57d82a30a // indirect
)
