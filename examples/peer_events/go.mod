module peer_events

go 1.15

require (
	github.com/ethereum/go-ethereum v1.9.5
	github.com/ipfs/go-log v1.0.5
	github.com/multiformats/go-multiaddr v0.3.1 // indirect
	github.com/status-im/go-waku v0.0.0-20210525134543-f6ebb5b0f9ae
)

replace github.com/status-im/go-waku => ../..

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.12
