package main

import (
	"context"
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func execute(options Options) {
	var err error
	hostAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("0.0.0.0:%d", options.Port))

	if options.NodeKey == nil {
		options.NodeKey, err = crypto.GenerateKey()
		if err != nil {
			fmt.Println("Could not generate random key")
			return
		}
	}

	connNotifier := make(chan node.PeerConnection)

	opts := []node.WakuNodeOption{
		node.WithPrivateKey(options.NodeKey),
		node.WithNTP(),
		node.WithHostAddress(hostAddr),
		node.WithConnectionNotification(connNotifier),
	}

	if options.Relay.Enable {
		opts = append(opts, node.WithWakuRelay())
	}

	if options.RLNRelay.Enable {
		spamHandler := func(message *pb.WakuMessage, topic string) error {
			return nil
		}

		if options.RLNRelay.Dynamic {
			fmt.Println("Setting up dynamic rln...")
			opts = append(opts, node.WithDynamicRLNRelay(
				options.RLNRelay.CredentialsPath,
				options.RLNRelay.CredentialsPassword,
				"", // Will use default tree path
				options.RLNRelay.MembershipContractAddress,
				options.RLNRelay.MembershipIndex,
				spamHandler,
				options.RLNRelay.ETHClientAddress,
			))
		} else {
			opts = append(opts, node.WithStaticRLNRelay(
				options.RLNRelay.MembershipIndex,
				spamHandler))
		}
	}

	if options.Filter.Enable {
		opts = append(opts, node.WithWakuFilterLightNode())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wakuNode, err := node.New(opts...)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = addPeer(wakuNode, options.Store.Node, options.Relay.Topics.Value(), legacy_store.StoreID_v20beta4)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = addPeer(wakuNode, options.LightPush.Node, options.Relay.Topics.Value(), lightpush.LightPushID_v20beta1)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = addPeer(wakuNode, options.Filter.Node, options.Relay.Topics.Value(), filter.FilterSubscribeID_v20beta1)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if err := wakuNode.Start(ctx); err != nil {
		fmt.Println(err.Error())
		return
	}

	chat := NewChat(ctx, wakuNode, connNotifier, options)
	p := tea.NewProgram(chat.ui)
	if err := p.Start(); err != nil {
		fmt.Println(err.Error())
	}

	cancel()

	wakuNode.Stop()
	chat.Stop()
}

func addPeer(wakuNode *node.WakuNode, addr *multiaddr.Multiaddr, topics []string, protocols ...protocol.ID) error {
	if addr == nil {
		return nil
	}
	_, err := wakuNode.AddPeer(*addr, peerstore.Static, topics, protocols...)
	return err
}
