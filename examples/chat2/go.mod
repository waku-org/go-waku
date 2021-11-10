module chat2

go 1.15

replace github.com/status-im/go-waku => ../..

require (
	github.com/ethereum/go-ethereum v1.10.9
	github.com/gdamore/tcell/v2 v2.2.0
	github.com/golang/protobuf v1.5.2
	github.com/ipfs/go-log v1.0.5
	github.com/libp2p/go-libp2p-core v0.9.0
	github.com/multiformats/go-multiaddr v0.4.0
	github.com/rivo/tview v0.0.0-20210312174852-ae9464cc3598
	github.com/status-im/go-waku v0.0.0-20211101194039-94e8b9cf86fc
	golang.org/x/crypto v0.0.0-20210813211128-0a44fdfbc16e
	google.golang.org/protobuf v1.27.1
)
