module basic2

go 1.15

replace github.com/status-im/go-waku => ../..

replace github.com/ethereum/go-ethereum v1.10.17 => github.com/status-im/go-ethereum v1.10.4-status.2

require (
	github.com/ethereum/go-ethereum v1.10.17
	github.com/ipfs/go-log v1.0.5
	github.com/status-im/go-waku v0.0.0-20211101194039-94e8b9cf86fc
)
