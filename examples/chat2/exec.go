package main

import (
	"context"
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/filterv2"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/utils"
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
		node.WithNTP(),
		node.WithHostAddress(hostAddr),
	}

	if options.Relay.Enable {
		opts = append(opts, node.WithWakuRelay())
	}

	if options.RLNRelay.Enable {
		spamHandler := func(message *pb.WakuMessage) error {
			return nil
		}

		registrationHandler := func(tx *types.Transaction) {
			fmt.Println(fmt.Sprintf("You are registered to the rln membership contract, find details of your registration transaction in https://goerli.etherscan.io/tx/%s", tx.Hash()))
		}

		if options.RLNRelay.Dynamic {
			membershipCredentials, err := node.GetMembershipCredentials(
				utils.Logger(),
				options.RLNRelay.CredentialsPath,
				options.RLNRelay.CredentialsPassword,
				options.RLNRelay.MembershipContractAddress,
				uint(options.RLNRelay.MembershipIndex),
			)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println("Setting up dynamic rln...")
			opts = append(opts, node.WithDynamicRLNRelay(
				options.RLNRelay.PubsubTopic,
				options.RLNRelay.ContentTopic,
				membershipCredentials,
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
			opts = append(opts, node.WithWakuFilterV2LightNode())
		} else {
			opts = append(opts, node.WithWakuFilter(false))
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wakuNode, err := node.New(opts...)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = addPeer(wakuNode, options.Store.Node, string(store.StoreID_v20beta4))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = addPeer(wakuNode, options.LightPush.Node, string(lightpush.LightPushID_v20beta1))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if options.Filter.UseV2 {
		err = addPeer(wakuNode, options.Filter.Node, string(filterv2.FilterSubscribeID_v20beta1))
	} else {
		err = addPeer(wakuNode, options.Filter.Node, string(filter.FilterID_v20beta1))
	}
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if err := wakuNode.Start(ctx); err != nil {
		fmt.Println(err.Error())
		return
	}

	if options.RLNRelay.Enable && options.RLNRelay.Dynamic {
		err := node.WriteRLNMembershipCredentialsToFile(wakuNode.RLNRelay().MembershipKeyPair(), wakuNode.RLNRelay().MembershipIndex(), wakuNode.RLNRelay().MembershipContractAddress(), options.RLNRelay.CredentialsPath, []byte(options.RLNRelay.CredentialsPassword))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	chat := NewChat(ctx, wakuNode, options)
	p := tea.NewProgram(chat.ui)
	if err := p.Start(); err != nil {
		fmt.Println(err.Error())
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
