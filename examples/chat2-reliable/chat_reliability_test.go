package main

import (
	"chat2-reliable/pb"
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

type TestEnvironment struct {
	nodes []*node.WakuNode
	chats []*Chat
}

func setupTestEnvironment(t *testing.T, nodeCount int) *TestEnvironment {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := &TestEnvironment{
		nodes: make([]*node.WakuNode, nodeCount),
		chats: make([]*Chat, nodeCount),
	}

	for i := 0; i < nodeCount; i++ {
		topics := cli.StringSlice{}
		topics.Set(relay.DefaultWakuTopic)

		options := Options{
			Port:         8000 + i, // Use different ports for each node
			Nickname:     fmt.Sprintf("Node%d", i),
			ContentTopic: "/test/1/chat/proto",
			Relay: RelayOptions{
				Enable: true,
				Topics: topics,
			},
		}

		hostAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("0.0.0.0:%d", options.Port))
		require.NoError(t, err)

		options.NodeKey, err = crypto.GenerateKey()
		require.NoError(t, err)

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

		wakuNode, err := node.New(opts...)
		require.NoError(t, err)

		env.nodes[i] = wakuNode

		chat := NewChat(ctx, wakuNode, connNotifier, options)
		env.chats[i] = chat

		err = wakuNode.Start(ctx)
		require.NoError(t, err)
	}

	// Connect nodes in a ring topology
	for i := 0; i < nodeCount; i++ {
		nextIndex := (i + 1) % nodeCount
		nextAddr := env.nodes[nextIndex].ListenAddresses()[0]
		_, err := env.nodes[i].AddPeer(nextAddr, peerstore.Static, env.chats[i].options.Relay.Topics.Value())
		require.NoError(t, err)
	}

	return env
}

// TestLamportTimestamps verifies that Lamport timestamps are correctly updated
func TestLamportTimestamps(t *testing.T) {
	env := setupTestEnvironment(t, 3)
	defer func() {
		for _, node := range env.nodes {
			node.Stop()
		}
	}()

	env.chats[0].SendMessage("Message from Node 0")
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), env.chats[0].lamportTimestamp)
	assert.Greater(t, env.chats[1].lamportTimestamp, int32(0))
	assert.Greater(t, env.chats[2].lamportTimestamp, int32(0))
}

// TestCausalOrdering ensures messages are processed in the correct causal order
func TestCausalOrdering(t *testing.T) {
	env := setupTestEnvironment(t, 3)
	defer func() {
		for _, node := range env.nodes {
			node.Stop()
		}
	}()

	env.chats[0].SendMessage("Message 1 from Node 0")
	time.Sleep(50 * time.Millisecond)
	env.chats[1].SendMessage("Message 2 from Node 1")
	time.Sleep(50 * time.Millisecond)
	env.chats[2].SendMessage("Message 3 from Node 2")
	time.Sleep(100 * time.Millisecond)

	for _, chat := range env.chats {
		assert.Len(t, chat.messageHistory, 3)
		assert.Equal(t, "Message 1 from Node 0", chat.messageHistory[0].Content)
		assert.Equal(t, "Message 2 from Node 1", chat.messageHistory[1].Content)
		assert.Equal(t, "Message 3 from Node 2", chat.messageHistory[2].Content)
	}
}

func TestBloomFilterDuplicateDetection(t *testing.T) {
	env := setupTestEnvironment(t, 2)

	msg := &pb.Message{
		SenderId:         "Node0",
		MessageId:        "test-message-id",
		LamportTimestamp: 1,
		Content:          "Test message",
	}

	// Send the message twice
	env.chats[0].processReceivedMessage(msg)
	env.chats[0].processReceivedMessage(msg)

	assert.Len(t, env.chats[0].messageHistory, 1)
}

func TestMessageRecovery(t *testing.T) {
	env := setupTestEnvironment(t, 3)

	// Send messages to establish causal history
	env.chats[0].SendMessage("Message 1")
	time.Sleep(50 * time.Millisecond)
	env.chats[1].SendMessage("Message 2")
	time.Sleep(50 * time.Millisecond)

	// Simulate a missed message
	missedMsg := &pb.Message{
		SenderId:         "Node2",
		MessageId:        "missed-message-id",
		LamportTimestamp: 3,
		CausalHistory:    []string{"Message 1", "Message 2"},
		Content:          "Missed Message",
	}

	// Only send to Node 0 and Node 1, simulating Node 2 missing the message
	env.chats[0].processReceivedMessage(missedMsg)
	env.chats[1].processReceivedMessage(missedMsg)

	// Send a new message from Node 2 that depends on the missed message
	newMsg := &pb.Message{
		SenderId:         "Node2",
		MessageId:        "new-message-id",
		LamportTimestamp: 4,
		CausalHistory:    []string{"missed-message-id"},
		Content:          "New Message",
	}

	env.chats[2].processReceivedMessage(newMsg)

	// Give some time for message recovery
	time.Sleep(200 * time.Millisecond)

	// Check if Node 2 has recovered the missed message
	assert.Len(t, env.chats[2].messageHistory, 4)
	assert.Equal(t, "Missed Message", env.chats[2].messageHistory[2].Content)
	assert.Equal(t, "New Message", env.chats[2].messageHistory[3].Content)
}

func TestConcurrentMessageSending(t *testing.T) {
	env := setupTestEnvironment(t, 5)

	messageCount := 10
	var wg sync.WaitGroup

	for i := 0; i < len(env.chats); i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < messageCount; j++ {
				env.chats[index].SendMessage(fmt.Sprintf("Message %d from Node %d", j, index))
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond) // Allow time for message propagation

	totalExpectedMessages := len(env.chats) * messageCount

	for i, chat := range env.chats {
		assert.Len(t, chat.messageHistory, totalExpectedMessages, "Node %d should have received all messages", i)
	}
}

func TestNetworkPartition(t *testing.T) {
	env := setupTestEnvironment(t, 4)

	// Send initial messages
	env.chats[0].SendMessage("Message 1 from Node 0")
	env.chats[1].SendMessage("Message 2 from Node 1")
	time.Sleep(100 * time.Millisecond)

	// Simulate network partition: disconnect Node 2 and Node 3
	env.nodes[2].Stop()
	env.nodes[3].Stop()

	// Send messages during partition
	env.chats[0].SendMessage("Message 3 from Node 0 during partition")
	env.chats[1].SendMessage("Message 4 from Node 1 during partition")
	time.Sleep(100 * time.Millisecond)

	// Reconnect Node 2 and Node 3
	env.nodes[2].Start(context.Background())
	env.nodes[3].Start(context.Background())

	// Allow time for synchronization
	time.Sleep(500 * time.Millisecond)

	// Verify all nodes have all messages
	for i, chat := range env.chats {
		assert.Len(t, chat.messageHistory, 4, "Node %d should have all messages after partition", i)
	}
}

func TestLargeGroupScaling(t *testing.T) {
	nodeCount := 20
	env := setupTestEnvironment(t, nodeCount)

	// Send a message from the first node
	env.chats[0].SendMessage("Broadcast message to large group")

	// Allow time for propagation
	time.Sleep(time.Duration(nodeCount*100) * time.Millisecond)

	// Verify all nodes received the message
	for i, chat := range env.chats {
		assert.Len(t, chat.messageHistory, 1, "Node %d should have received the broadcast message", i)
		assert.Equal(t, "Broadcast message to large group", chat.messageHistory[0].Content)
	}
}
