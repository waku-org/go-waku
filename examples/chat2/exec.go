package main

import (
	"context"
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-waku/waku/v2/protocol/filter"
	"github.com/status-im/go-waku/waku/v2/protocol/lightpush"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
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

	opts := []node.WakuNodeOption{
		node.WithPrivateKey(options.NodeKey),
		node.WithHostAddress(hostAddr),
		node.WithWakuStore(false, false),
	}

	if options.Relay.Enable {
		opts = append(opts, node.WithWakuRelay())
	}

	if options.RLNRelay.Enable {
		spamHandler := func(message *pb.WakuMessage) error {
			return nil
		}

		if options.RLNRelay.Dynamic {
			idKey, idCommitment, index, err := getMembershipCredentials(options.RLNRelay.CredentialsFile, options.RLNRelay.IDKey, options.RLNRelay.IDCommitment, options.RLNRelay.MembershipIndex)
			if err != nil {
				panic(err)
			}

			fmt.Println("Setting up dynamic rln")
			opts = append(opts, node.WithDynamicRLNRelay(
				options.RLNRelay.PubsubTopic,
				options.RLNRelay.ContentTopic,
				index,
				idKey,
				idCommitment,
				spamHandler,
				options.RLNRelay.ETHClientAddress,
				options.RLNRelay.ETHPrivateKey,
				options.RLNRelay.MembershipContractAddress,
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
		opts = append(opts, node.WithWakuFilter(false))
	}

	if options.LightPush.Enable {
		opts = append(opts, node.WithLightPush())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wakuNode, err := node.New(ctx, opts...)
	if err != nil {
		fmt.Print(err)
		return
	}

	err = addPeer(wakuNode, options.Store.Node, string(store.StoreID_v20beta4))
	if err != nil {
		fmt.Print(err)
		return
	}

	err = addPeer(wakuNode, options.LightPush.Node, string(lightpush.LightPushID_v20beta1))
	if err != nil {
		fmt.Print(err)
		return
	}

	err = addPeer(wakuNode, options.Filter.Node, string(filter.FilterID_v20beta1))
	if err != nil {
		fmt.Print(err)
		return
	}

	if err := wakuNode.Start(); err != nil {
		panic(err)
	}

	if options.RLNRelay.Enable && options.RLNRelay.Dynamic {
		if options.RLNRelay.IDKey == "" && options.RLNRelay.IDCommitment == "" {
			// Write membership credentials file only if the idkey and commitment are not specified
			err := writeRLNMembershipCredentialsToFile(options.RLNRelay.CredentialsFile, wakuNode.RLNRelay().MembershipKeyPair(), wakuNode.RLNRelay().MembershipIndex())
			if err != nil {
				panic(err)
			}
		}
	}

	chat := NewChat(ctx, wakuNode, options)
	p := tea.NewProgram(chat.ui)
	if err := p.Start(); err != nil {
		fmt.Println(err)
	}

	cancel()

	wakuNode.Stop()
	chat.Stop()
}

func addPeer(wakuNode *node.WakuNode, addr *multiaddr.Multiaddr, protocols ...string) error {
	if addr == nil {
		return nil
	}
	_, err := wakuNode.AddPeer(*addr, protocols...)
	return err
}
