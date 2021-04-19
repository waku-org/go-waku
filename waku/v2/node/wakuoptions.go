package node

import (
	"context"
	"crypto/ecdsa"
	"net"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	store "github.com/status-im/go-waku/waku/v2/protocol/waku_store"
	wakurelay "github.com/status-im/go-wakurelay-pubsub"
)

type WakuNodeParameters struct {
	multiAddr  []ma.Multiaddr
	privKey    *crypto.PrivKey
	libP2POpts []libp2p.Option

	enableRelay bool
	wOpts       []wakurelay.Option

	enableStore bool
	storeMsgs   bool
	store       *store.WakuStore

	ctx context.Context
}

type WakuNodeOption func(*WakuNodeParameters) error

func WithHostAddress(hostAddr []net.Addr) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		var multiAddresses []ma.Multiaddr
		for _, addr := range hostAddr {
			hostAddrMA, err := manet.FromNetAddr(addr)
			if err != nil {
				return err
			}
			multiAddresses = append(multiAddresses, hostAddrMA)
		}

		params.multiAddr = multiAddresses

		return nil
	}
}

func WithPrivateKey(privKey *ecdsa.PrivateKey) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		privk := crypto.PrivKey((*crypto.Secp256k1PrivateKey)(privKey))
		params.privKey = &privk
		return nil
	}
}

func WithLibP2POptions(opts ...libp2p.Option) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.libP2POpts = opts
		return nil
	}
}

func WithWakuRelay(opts ...wakurelay.Option) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRelay = true
		params.wOpts = opts
		return nil
	}
}

func WithWakuStore(shouldStoreMessages bool) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableStore = true
		params.storeMsgs = shouldStoreMessages
		params.store = store.NewWakuStore(params.ctx, shouldStoreMessages, nil)
		return nil
	}
}

func WithMessageProvider(s store.MessageProvider) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		if params.store != nil {
			params.store.SetMsgProvider(s)
		} else {
			params.store = store.NewWakuStore(params.ctx, true, s)
		}
		return nil
	}
}

var DefaultLibP2POptions = []libp2p.Option{
	libp2p.DefaultTransports,
	libp2p.NATPortMap(),       // Attempt to open ports using uPNP for NATed hosts.
	libp2p.EnableNATService(), // TODO: is this needed?)
	libp2p.ConnectionManager(connmgr.NewConnManager(200, 300, 0)),
}
