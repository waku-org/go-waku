package main

import (
	"context"
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/peers"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter"
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
		spamHandler := func(message *pb.WakuMessage) error {
			return nil
		}

		registrationHandler := func(tx *types.Transaction) {
			chainID := tx.ChainId().Int64()
			url := ""
			switch chainID {
			case 1:
				url = "https://etherscan.io"
			case 5:
				url = "https://goerli.etherscan.io"
			case 11155111:
				url = "https://sepolia.etherscan.io"

			}

			if url != "" {
				fmt.Println(fmt.Sprintf("You are registered to the rln membership contract, find details of your registration transaction in %s/tx/%s", url, tx.Hash()))
			} else {
				fmt.Println(fmt.Sprintf("You are registered to the rln membership contract. Transaction hash: %s", url, tx.Hash()))
			}
		}

		if options.RLNRelay.Dynamic {
			fmt.Println("Setting up dynamic rln...")
			opts = append(opts, node.WithDynamicRLNRelay(
				options.RLNRelay.PubsubTopic,
				options.RLNRelay.ContentTopic,
				options.RLNRelay.CredentialsPath,
				options.RLNRelay.CredentialsPassword,
				options.RLNRelay.CredentialsIndex,
				options.RLNRelay.MembershipContractAddress,
				uint(options.RLNRelay.MembershipIndex),
				spamHandler,
				options.RLNRelay.ETHClientAddress,
				options.RLNRelay.ETHPrivateKey,
				registrationHandler,
			))
		} else {
			opts = append(opts, node.WithStaticRLNRelay(
				options.RLNRelay.PubsubTopic,
				options.RLNRelay.ContentTopic,
				uint(options.RLNRelay.MembershipIndex),
				spamHandler))
		}
	}

	if options.Filter.Enable {
		if options.Filter.UseV2 {
			opts = append(opts, node.WithWakuFilterLightNode())
		} else {
			opts = append(opts, node.WithLegacyWakuFilter(false))
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wakuNode, err := node.New(opts...)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = addPeer(wakuNode, options.Store.Node, store.StoreID_v20beta4)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = addPeer(wakuNode, options.LightPush.Node, lightpush.LightPushID_v20beta1)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if options.Filter.UseV2 {
		err = addPeer(wakuNode, options.Filter.Node, filter.FilterSubscribeID_v20beta1)
	} else {
		err = addPeer(wakuNode, options.Filter.Node, legacy_filter.FilterID_v20beta1)
	}
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

func addPeer(wakuNode *node.WakuNode, addr *multiaddr.Multiaddr, protocols ...protocol.ID) error {
	if addr == nil {
		return nil
	}
	_, err := wakuNode.AddPeer(*addr, peers.Static, protocols...)
	return err
}
