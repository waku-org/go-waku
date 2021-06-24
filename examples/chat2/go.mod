module chat2

go 1.15

replace github.com/status-im/go-waku => ../..

require (
	github.com/ethereum/go-ethereum v1.10.1
	github.com/gdamore/tcell/v2 v2.2.0
	github.com/golang/protobuf v1.5.2
	github.com/ipfs/go-log v1.0.4
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/rivo/tview v0.0.0-20210312174852-ae9464cc3598
	github.com/status-im/go-waku v0.0.0-20210428201044-3d8aae5b81b9
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	google.golang.org/protobuf v1.26.0
)
