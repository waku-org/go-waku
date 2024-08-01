package main

import (
	"chat2-reliable/pb"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

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

func setupTestEnvironment(ctx context.Context, t *testing.T, nodeCount int) (*TestEnvironment, error) {
	t.Logf("Setting up test environment with %d nodes", nodeCount)
	env := &TestEnvironment{
		nodes: make([]*node.WakuNode, nodeCount),
		chats: make([]*Chat, nodeCount),
	}

	for i := 0; i < nodeCount; i++ {
		t.Logf("Setting up node %d", i)
		node, err := setupTestNode(ctx, t, i)
		if err != nil {
			return nil, fmt.Errorf("failed to set up node %d: %w", i, err)
		}
		env.nodes[i] = node

		t.Logf("Creating chat instance for node %d", i)
		chat, err := setupTestChat(ctx, t, node, fmt.Sprintf("Node%d", i))
		if err != nil {
			return nil, fmt.Errorf("failed to set up chat for node %d: %w", i, err)
		}
		env.chats[i] = chat
	}

	t.Log("Connecting nodes in ring topology")
	for i := 0; i < nodeCount; i++ {
		nextIndex := (i + 1) % nodeCount
		t.Logf("Connecting node %d to node %d", i, nextIndex)
		_, err := env.nodes[i].AddPeer(env.nodes[nextIndex].ListenAddresses()[0], peerstore.Static, env.chats[i].options.Relay.Topics.Value())
		if err != nil {
			return nil, fmt.Errorf("failed to connect node %d to node %d: %w", i, nextIndex, err)
		}
	}

	t.Log("Test environment setup complete")
	return env, nil
}

func setupTestNode(ctx context.Context, t *testing.T, index int) (*node.WakuNode, error) {
	opts := []node.WakuNodeOption{
		node.WithWakuRelay(),
		// node.WithWakuStore(),
	}
	node, err := node.New(opts...)
	if err != nil {
		return nil, err
	}
	if err := node.Start(ctx); err != nil {
		return nil, err
	}

	// if node.Store() == nil {
	// 	t.Logf("Store protocol is not enabled on node %d", index)
	// }

	return node, nil
}

type PeerConnection = node.PeerConnection

func setupTestChat(ctx context.Context, t *testing.T, node *node.WakuNode, nickname string) (*Chat, error) {
	topics := cli.StringSlice{}
	topics.Set(relay.DefaultWakuTopic)

	options := Options{
		Nickname:     nickname,
		ContentTopic: "/test/1/chat/proto",
		Relay: RelayOptions{
			Enable: true,
			Topics: topics,
		},
	}

	// Create a channel of the correct type
	connNotifier := make(chan PeerConnection)

	chat := NewChat(ctx, node, connNotifier, options)
	if chat == nil {
		return nil, fmt.Errorf("failed to create chat instance")
	}
	return chat, nil
}

// TestLamportTimestamps verifies that Lamport timestamps are correctly updated
func TestLamportTimestamps(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Log("Starting TestLamportTimestamps")

	nodeCount := 3
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")

	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, nodeCount)
	}, 30*time.Second, 1*time.Second, "Nodes failed to connect")

	for i, chat := range env.chats {
		t.Logf("Node %d initial Lamport timestamp: %d", i, chat.getLamportTimestamp())
	}

	t.Log("Sending message from Node 0")
	env.chats[0].SendMessage("Message from Node 0")

	t.Log("Waiting for message propagation")
	require.Eventually(t, func() bool {
		for _, chat := range env.chats {
			if chat.getLamportTimestamp() == 0 {
				return false
			}
		}
		return true
	}, 30*time.Second, 1*time.Second, "Message propagation failed")

	assert.Equal(t, int32(1), env.chats[0].getLamportTimestamp(), "Sender's Lamport timestamp should be 1")
	assert.Greater(t, env.chats[1].getLamportTimestamp(), int32(0), "Node 1's Lamport timestamp should be greater than 0")
	assert.Greater(t, env.chats[2].getLamportTimestamp(), int32(0), "Node 2's Lamport timestamp should be greater than 0")

	assert.NotEmpty(t, env.chats[1].messageHistory, "Node 1 should have received the message")
	assert.NotEmpty(t, env.chats[2].messageHistory, "Node 2 should have received the message")

	if len(env.chats[1].messageHistory) > 0 {
		assert.Equal(t, "Message from Node 0", env.chats[1].messageHistory[0].Content, "Node 1 should have received the correct message")
	}
	if len(env.chats[2].messageHistory) > 0 {
		assert.Equal(t, "Message from Node 0", env.chats[2].messageHistory[0].Content, "Node 2 should have received the correct message")
	}

	t.Log("TestLamportTimestamps completed successfully")
}

// TestCausalOrdering ensures messages are processed in the correct causal order
func TestCausalOrdering(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Log("Starting TestCausalOrdering")

	nodeCount := 3
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")

	// defer tearDownEnvironment(t, env)

	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, nodeCount)
	}, 30*time.Second, 1*time.Second, "Nodes failed to connect")

	t.Log("Sending messages from different nodes")
	env.chats[0].SendMessage("Message 1 from Node 0")
	time.Sleep(100 * time.Millisecond)
	env.chats[1].SendMessage("Message 2 from Node 1")
	time.Sleep(100 * time.Millisecond)
	env.chats[2].SendMessage("Message 3 from Node 2")
	time.Sleep(100 * time.Millisecond)

	t.Log("Waiting for message propagation")
	require.Eventually(t, func() bool {
		for i, chat := range env.chats {
			t.Logf("Node %d message history length: %d", i, len(chat.messageHistory))
			if len(chat.messageHistory) != 3 {
				return false
			}
		}
		return true
	}, 30*time.Second, 1*time.Second, "Messages did not propagate to all nodes")

	for i, chat := range env.chats {
		assert.Len(t, chat.messageHistory, 3, "Node %d should have 3 messages", i)
		assert.Equal(t, "Message 1 from Node 0", chat.messageHistory[0].Content, "Node %d: First message incorrect", i)
		assert.Equal(t, "Message 2 from Node 1", chat.messageHistory[1].Content, "Node %d: Second message incorrect", i)
		assert.Equal(t, "Message 3 from Node 2", chat.messageHistory[2].Content, "Node %d: Third message incorrect", i)
	}

	t.Log("TestCausalOrdering completed successfully")
}

func TestBloomFilterDuplicateDetection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Log("Starting TestBloomFilterDuplicateDetection")

	nodeCount := 2
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")

	//defer tearDownEnvironment(t, env)

	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, nodeCount)
	}, 30*time.Second, 1*time.Second, "Nodes failed to connect")

	t.Log("Sending a message")
	testMessage := "Test message"
	env.chats[0].SendMessage(testMessage)

	t.Log("Waiting for message propagation")
	var receivedMsg *pb.Message
	require.Eventually(t, func() bool {
		if len(env.chats[1].messageHistory) == 1 {
			receivedMsg = env.chats[1].messageHistory[0]
			return true
		}
		return false
	}, 30*time.Second, 1*time.Second, "Message did not propagate to second node")

	require.NotNil(t, receivedMsg, "Received message should not be nil")

	t.Log("Simulating receiving the same message again")

	// Create a duplicate message
	duplicateMsg := &pb.Message{
		SenderId:         receivedMsg.SenderId,
		MessageId:        receivedMsg.MessageId, // Use the same MessageId to simulate a true duplicate
		LamportTimestamp: receivedMsg.LamportTimestamp,
		CausalHistory:    receivedMsg.CausalHistory,
		ChannelId:        receivedMsg.ChannelId,
		BloomFilter:      receivedMsg.BloomFilter,
		Content:          receivedMsg.Content,
	}

	env.chats[1].processReceivedMessage(duplicateMsg)

	assert.Len(t, env.chats[1].messageHistory, 1, "Node 1 should still have only one message (no duplicates)")

	t.Log("TestBloomFilterDuplicateDetection completed successfully")
}

func TestMessageRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Log("Starting TestMessageRecovery")

	nodeCount := 3
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")
	//defer tearDownEnvironment(t, env)

	nc := NewNetworkController(env.nodes)

	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, nodeCount)
	}, 60*time.Second, 1*time.Second, "Nodes failed to connect")

	// Stage 1: Send initial messages
	t.Log("Stage 1: Sending initial messages")
	env.chats[0].SendMessage("Message 1")
	time.Sleep(100 * time.Millisecond)
	env.chats[1].SendMessage("Message 2")
	time.Sleep(100 * time.Millisecond)

	t.Log("Waiting for message propagation")
	require.Eventually(t, func() bool {
		for i, chat := range env.chats {
			t.Logf("Node %d message history length: %d", i, len(chat.messageHistory))
			if len(chat.messageHistory) != 2 {
				return false
			}
		}
		return true
	}, 30*time.Second, 1*time.Second, "Messages did not propagate to all nodes")

	// Verify that Node 2 has messages before disconnecion
	require.Equal(t, 2, len(env.chats[2].messageHistory), "Node 2 does not have all messages")

	// Stage 2: Simulate network partition
	t.Log("Stage 2: Simulating network partition for Node 2")
	nc.DisconnectNode(env.nodes[2])
	time.Sleep(1 * time.Second) // Allow time for disconnection to take effect

	// Stage 3: Send message that Node 2 will miss
	t.Log("Stage 3: Sending message that Node 2 will miss")
	env.chats[0].SendMessage("Missed Message")
	time.Sleep(2 * time.Second)

	// Stage 4: Reconnect Node 2
	t.Log("Stage 4: Reconnecting Node 2")
	nc.ReconnectNode(env.nodes[2])
	time.Sleep(5 * time.Second) // Allow time for reconnection to take effect

	// Verify that Node 2 didn't receive the message
	require.Equal(t, 2, len(env.chats[2].messageHistory), "Node 2 should not have received the missed message")

	// Stage 5: Send a new message that depends on the missed message
	t.Log("Stage 5: Sending a new message that depends on the missed message")
	env.chats[1].SendMessage("New Message")
	time.Sleep(5 * time.Second)

	// Verify that Node 2 received the new message
	require.Equal(t, 3, len(env.chats[2].messageHistory), "Node 2 should have received the new message")

	// Stage 6: Wait for message recovery
	t.Log("Stage 6: Waiting for message recovery")
	require.Eventually(t, func() bool {
		msgCount := len(env.chats[2].messageHistory)
		t.Logf("Disconnected node msg count: %d", msgCount)
		return msgCount == 4
	}, 60*time.Second, 5*time.Second, "Message recovery failed")

	// Verify the results
	for i, msg := range env.chats[2].messageHistory {
		t.Logf("Message %d: %s", i+1, msg.Content)
	}

	assert.Equal(t, "Message 1", env.chats[2].messageHistory[0].Content, "First message incorrect")
	assert.Equal(t, "Message 2", env.chats[2].messageHistory[1].Content, "Second message incorrect")
	assert.Equal(t, "Missed Message", env.chats[2].messageHistory[2].Content, "Missed message not recovered")
	assert.Equal(t, "New Message", env.chats[2].messageHistory[3].Content, "New message incorrect")
}

func TestConcurrentMessageSending(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Log("Starting TestConcurrentMessageSending")

	nodeCount := 5
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")

	//defer tearDownEnvironment(t, env)

	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, nodeCount)
	}, 30*time.Second, 1*time.Second, "Nodes failed to connect")

	messageCount := 10
	var wg sync.WaitGroup

	t.Log("Sending messages concurrently")
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

	t.Log("Waiting for message propagation")
	totalExpectedMessages := len(env.chats) * messageCount
	require.Eventually(t, func() bool {
		for _, chat := range env.chats {
			if len(chat.messageHistory) != totalExpectedMessages {
				return false
			}
		}
		return true
	}, 2*time.Minute, 1*time.Second, "Messages did not propagate to all nodes")

	for i, chat := range env.chats {
		assert.Len(t, chat.messageHistory, totalExpectedMessages, "Node %d should have received all messages", i)
	}

	t.Log("TestConcurrentMessageSending completed successfully")
}

// Helper functions

func tearDownEnvironment(t *testing.T, env *TestEnvironment) {
	t.Log("Tearing down test environment")
	for i, node := range env.nodes {
		t.Logf("Stopping node %d", i)
		node.Stop()
	}
	for i, chat := range env.chats {
		t.Logf("Stopping chat %d", i)
		chat.Stop()
	}
}

func areNodesConnected(nodes []*node.WakuNode, expectedConnections int) bool {
	for _, node := range nodes {
		if len(node.Host().Network().Peers()) != expectedConnections-1 {
			return false
		}
	}
	return true
}

// func TestNetworkPartition(t *testing.T) {
// 	env := setupTestEnvironment(t, 4)

// 	// Send initial messages
// 	env.chats[0].SendMessage("Message 1 from Node 0")
// 	env.chats[1].SendMessage("Message 2 from Node 1")
// 	time.Sleep(100 * time.Millisecond)

// 	// Simulate network partition: disconnect Node 2 and Node 3
// 	env.nodes[2].Stop()
// 	env.nodes[3].Stop()

// 	// Send messages during partition
// 	env.chats[0].SendMessage("Message 3 from Node 0 during partition")
// 	env.chats[1].SendMessage("Message 4 from Node 1 during partition")
// 	time.Sleep(100 * time.Millisecond)

// 	// Reconnect Node 2 and Node 3
// 	env.nodes[2].Start(context.Background())
// 	env.nodes[3].Start(context.Background())

// 	// Allow time for synchronization
// 	time.Sleep(500 * time.Millisecond)

// 	// Verify all nodes have all messages
// 	for i, chat := range env.chats {
// 		assert.Len(t, chat.messageHistory, 4, "Node %d should have all messages after partition", i)
// 	}
// }

// func TestLargeGroupScaling(t *testing.T) {
// 	nodeCount := 20
// 	env := setupTestEnvironment(t, nodeCount)

// 	// Send a message from the first node
// 	env.chats[0].SendMessage("Broadcast message to large group")

// 	// Allow time for propagation
// 	time.Sleep(time.Duration(nodeCount*100) * time.Millisecond)

// 	// Verify all nodes received the message
// 	for i, chat := range env.chats {
// 		assert.Len(t, chat.messageHistory, 1, "Node %d should have received the broadcast message", i)
// 		assert.Equal(t, "Broadcast message to large group", chat.messageHistory[0].Content)
// 	}
// }
