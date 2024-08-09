package main

import (
	"context"
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
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

	if options.DiscV5.Enable {
		nodes := []*enode.Node{}
		for _, n := range options.DiscV5.Nodes.Value() {
			parsedNode, err := enode.Parse(enode.ValidSchemes, n)
			if err != nil {
				fmt.Println("Failed to parse DiscV5 node ", err)
				return
			}
			nodes = append(nodes, parsedNode)
		}
		opts = append(opts, node.WithDiscoveryV5(uint(options.DiscV5.Port), nodes, options.DiscV5.AutoUpdate))
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

	if err := wakuNode.Start(ctx); err != nil {
		fmt.Println(err.Error())
		return
	}

	err = addPeer(wakuNode, options.Store.Node, options.Relay.Topics.Value(), store.StoreQueryID_v300)
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
