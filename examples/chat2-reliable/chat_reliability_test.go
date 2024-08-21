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
		node, err := setupTestNode(ctx, t)
		if err != nil {
			return nil, fmt.Errorf("failed to set up node %d: %w", i, err)
		}
		env.nodes[i] = node

		chat, err := setupTestChat(ctx, node, fmt.Sprintf("Node%d", i))
		if err != nil {
			return nil, fmt.Errorf("failed to set up chat for node %d: %w", i, err)
		}
		env.chats[i] = chat
	}

	t.Log("Connecting nodes in ring topology")
	for i := 0; i < nodeCount; i++ {
		nextIndex := (i + 1) % nodeCount
		_, err := env.nodes[i].AddPeer(env.nodes[nextIndex].ListenAddresses()[0], peerstore.Static, env.chats[i].options.Relay.Topics.Value())
		if err != nil {
			return nil, fmt.Errorf("failed to connect node %d to node %d: %w", i, nextIndex, err)
		}
	}

	t.Log("Test environment setup complete")
	return env, nil
}

func setupTestNode(ctx context.Context, t *testing.T) (*node.WakuNode, error) {
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

func setupTestChat(ctx context.Context, node *node.WakuNode, nickname string) (*Chat, error) {
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

func areNodesConnected(nodes []*node.WakuNode, expectedPeers int) bool {
	for _, node := range nodes {
		if len(node.Host().Network().Peers()) != expectedPeers {
			return false
		}
	}
	return true
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
		return areNodesConnected(env.nodes, 2)
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

	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, 2)
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

	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, 1)
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

// TestNetworkPartition ensures that missing messages can be recovered
func TestNetworkPartition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Log("Starting TestMessageRecovery")

	nodeCount := 3
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")

	nc := NewNetworkController(ctx, env.nodes, env.chats)

	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, 2)
	}, 60*time.Second, 1*time.Second, "Nodes failed to connect")

	t.Log("Stage 1: Sending initial messages")
	env.chats[0].SendMessage("Message 1")
	time.Sleep(100 * time.Millisecond)
	env.chats[1].SendMessage("Message 2")
	time.Sleep(100 * time.Millisecond)

	t.Log("Waiting for message propagation")
	require.Eventually(t, func() bool {
		for _, chat := range env.chats {
			if len(chat.messageHistory) != 2 {
				return false
			}
		}
		return true
	}, 30*time.Second, 1*time.Second, "Messages did not propagate to all nodes")

	// Verify that Node 2 has messages before disconnection
	require.Equal(t, 2, len(env.chats[2].messageHistory), "Node 2 does not have all messages")

	t.Log("Stage 2: Simulating network partition for Node 2")
	nc.DisconnectNode(env.nodes[2])
	time.Sleep(1 * time.Second) // Allow time for disconnection to take effect

	t.Log("Stage 3: Sending message that Node 2 will miss")
	env.chats[0].SendMessage("Missed Message")
	time.Sleep(100 * time.Millisecond)

	t.Log("Stage 4: Reconnecting Node 2")
	nc.ReconnectNode(env.nodes[2])
	time.Sleep(5 * time.Second) // Allow time for reconnection to take effect

	// Verify that Node 2 didn't receive the message
	require.Equal(t, 2, len(env.chats[2].messageHistory), "Node 2 should not have received the missed message")

	t.Log("Stage 5: Sending a new message that depends on the missed message")
	env.chats[1].SendMessage("New Message")

	// Verify that Node 2 received the new message
	require.Eventually(t, func() bool {
		msgCount := len(env.chats[2].messageHistory)
		return msgCount >= 3
	}, 30*time.Second, 5*time.Second, "Node 2 should have received the new message")

	// Stage 6: Wait for message recovery
	t.Log("Stage 6: Waiting for message recovery")
	require.Eventually(t, func() bool {
		msgCount := len(env.chats[2].messageHistory)
		return msgCount == 4
	}, 30*time.Second, 5*time.Second, "Message recovery failed")

	// Print final message history for all nodes
	for i, chat := range env.chats {
		t.Logf("Node %d final message history:", i)
		for j, msg := range chat.messageHistory {
			t.Logf("  Message %d: %s", j+1, msg.Content)
		}
	}

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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Log("Starting TestConcurrentMessageSending")

	nodeCount := 5
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")

	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, 2)
	}, 60*time.Second, 3*time.Second, "Nodes failed to connect")

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

func TestLargeGroupScaling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Log("Starting TestLargeGroupScaling")

	nodeCount := 20
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")

	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, 2)
	}, 2*time.Minute, 3*time.Second, "Nodes failed to connect")

	// Send a message from the first node
	env.chats[0].SendMessage("Broadcast message to large group")

	// Allow time for propagation
	time.Sleep(time.Duration(nodeCount*100) * time.Millisecond)

	// Verify all nodes received the message
	for i, chat := range env.chats {
		assert.Len(t, chat.messageHistory, 1, "Node %d should have received the broadcast message", i)
		assert.Equal(t, "Broadcast message to large group", chat.messageHistory[0].Content)
	}

	t.Log("TestLargeGroupScaling completed successfully")
}

func TestEagerPushMechanism(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	nodeCount := 2
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")

	nc := NewNetworkController(ctx, env.nodes, env.chats)

	// Disconnect node 1
	nc.DisconnectNode(env.nodes[1])

	// Send a message from node 0
	env.chats[0].SendMessage("Test eager push")

	// Wait for the message to be added to the outgoing buffer
	time.Sleep(1 * time.Second)

	// Reconnect node 1
	nc.ReconnectNode(env.nodes[1])

	// Wait for eager push to resend the message
	time.Sleep(5 * time.Second)

	// Check if node 1 received the message
	assert.Eventually(t, func() bool {
		return len(env.chats[1].messageHistory) == 1
	}, 10*time.Second, 1*time.Second, "Node 1 should have received the message via eager push")
}

func TestBloomFilterWindow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	nodeCount := 2
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")

	// Reduce bloom filter window for testing
	for _, chat := range env.chats {
		chat.bloomFilter.window = 2 * time.Second
	}

	// Send a message
	env.chats[0].SendMessage("Test bloom filter window")
	messageID := env.chats[0].messageHistory[0].MessageId

	// Check if the message is in the bloom filter
	assert.Eventually(t, func() bool {
		return env.chats[1].bloomFilter.Test(messageID)
	}, 30*time.Second, 1*time.Second, "Message should be in the bloom filter")

	// Wait for the bloom filter window to pass
	time.Sleep(3 * time.Second)

	// Clean the bloom filter
	env.chats[1].bloomFilter.Clean()

	time.Sleep(3 * time.Second)

	// Check if the message is no longer in the bloom filter
	assert.False(t, env.chats[1].bloomFilter.Test(messageID), "Message should no longer be in the bloom filter")

	// Send another message to ensure the filter still works for new messages
	env.chats[0].SendMessage("New test message")
	time.Sleep(1 * time.Second)

	newMessageID := env.chats[0].messageHistory[1].MessageId
	// Check if the new message is in the bloom filter
	assert.Eventually(t, func() bool {
		return env.chats[1].bloomFilter.Test(newMessageID)
	}, 30*time.Second, 1*time.Second, "New message should be in the bloom filter")
}

func TestConflictResolution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	nodeCount := 3
	env, err := setupTestEnvironment(ctx, t, nodeCount)
	require.NoError(t, err, "Failed to set up test environment")

	// Create conflicting messages with the same Lamport timestamp
	conflictingMsg1 := &pb.Message{
		SenderId:         "Node0",
		MessageId:        "msg1",
		LamportTimestamp: 1,
		Content:          "Conflict 1",
	}
	conflictingMsg2 := &pb.Message{
		SenderId:         "Node1",
		MessageId:        "msg2",
		LamportTimestamp: 1,
		Content:          "Conflict 2",
	}

	// Process the conflicting messages in different orders on different nodes
	env.chats[0].processReceivedMessage(conflictingMsg1)
	env.chats[0].processReceivedMessage(conflictingMsg2)

	env.chats[1].processReceivedMessage(conflictingMsg2)
	env.chats[1].processReceivedMessage(conflictingMsg1)

	// Check if the messages are ordered consistently across nodes
	assert.Equal(t, env.chats[0].messageHistory[0].MessageId, env.chats[1].messageHistory[0].MessageId, "Conflicting messages should be ordered consistently")
	assert.Equal(t, env.chats[0].messageHistory[1].MessageId, env.chats[1].messageHistory[1].MessageId, "Conflicting messages should be ordered consistently")
}

func TestNewNodeSyncAndMessagePropagation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Log("Starting TestNewNodeSyncAndMessagePropagation")

	// Set up initial network with 2 nodes
	initialNodeCount := 2
	env, err := setupTestEnvironment(ctx, t, initialNodeCount)
	require.NoError(t, err, "Failed to set up initial test environment")

	// Ensure initial nodes are connected
	require.Eventually(t, func() bool {
		return areNodesConnected(env.nodes, 1)
	}, 60*time.Second, 1*time.Second, "Initial nodes failed to connect")

	t.Log("Sending initial messages")
	env.chats[0].SendMessage("Initial message 1")
	env.chats[1].SendMessage("Initial message 2")

	// Wait for message propagation
	time.Sleep(5 * time.Second)

	// Verify initial messages are received by both nodes
	for i, chat := range env.chats {
		assert.Len(t, chat.messageHistory, 2, "Node %d should have 2 initial messages", i)
	}

	t.Log("Adding new node to the network")
	newNode, err := setupTestNode(ctx, t)
	require.NoError(t, err, "Failed to set up new node")
	newChat, err := setupTestChat(ctx, newNode, "NewNode")
	require.NoError(t, err, "Failed to set up new chat")

	env.nodes = append(env.nodes, newNode)
	env.chats = append(env.chats, newChat)

	// Connect new node to the network
	_, err = env.nodes[2].AddPeer(env.nodes[0].ListenAddresses()[0], peerstore.Static, env.chats[2].options.Relay.Topics.Value())
	require.NoError(t, err, "Failed to connect new node to the network")

	t.Log("Waiting for new node to sync")
	require.Eventually(t, func() bool {
		msgCount := len(env.chats[2].messageHistory)
		return msgCount == 2
	}, 1*time.Minute, 5*time.Second, "New node failed to sync message history")

	t.Log("Sending message from old node")
	env.chats[0].SendMessage("Message from old node")

	// Wait for message propagation
	time.Sleep(10 * time.Second)

	// Verify the message is received by all nodes
	for i, chat := range env.chats {
		assert.Len(t, chat.messageHistory, 3, "Node %d should have 3 messages", i)
	}

	t.Log("Sending message from new node")
	env.chats[2].SendMessage("Message from new node")

	// Wait for message propagation
	time.Sleep(10 * time.Second)

	// Verify the message from new node is received by all nodes
	for i, chat := range env.chats {
		assert.Len(t, chat.messageHistory, 4, "Node %d should have 4 messages", i)
	}

	for i := 0; i < 3; i++ {
		lastMsg := env.chats[i].messageHistory[len(env.chats[i].messageHistory)-1]
		assert.Equal(t, "Message from new node", lastMsg.Content, "The last message is incorrect for node %d", i)
	}

	t.Log("TestNewNodeSyncAndMessagePropagation completed")
}
