package pb

//go:generate protoc -I. --go_opt=paths=source_relative --go_opt=Mwaku_swap.proto=github.com/waku-org/go-waku/waku/v2/protocol/swap/pb --go_out=. ./waku_swap.proto
