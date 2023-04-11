package legacy_filter

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/test"
	"github.com/stretchr/testify/assert"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter/pb"
)

const TOPIC = "/test/topic"

func createPeerId(t *testing.T) peer.ID {
	peerId, err := test.RandPeerID()
	assert.NoError(t, err)
	return peerId
}

func firstSubscriber(subs *Subscribers, contentTopic string) *Subscriber {
	for sub := range subs.Items(&contentTopic) {
		return &sub
	}
	return nil
}

func TestAppend(t *testing.T) {
	subs := NewSubscribers(10 * time.Second)
	peerId := createPeerId(t)
	requestId := "request_1"
	contentTopic := "topic1"
	request := &pb.FilterRequest{
		Subscribe:      true,
		Topic:          TOPIC,
		ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: contentTopic}},
	}
	subs.Append(Subscriber{peerId, requestId, request})

	sub := firstSubscriber(subs, contentTopic)
	assert.NotNil(t, sub)
}

func TestRemove(t *testing.T) {
	subs := NewSubscribers(10 * time.Second)
	peerId := createPeerId(t)
	requestId := "request_1"
	contentTopic := "topic1"
	request := &pb.FilterRequest{
		Subscribe:      true,
		Topic:          TOPIC,
		ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: contentTopic}},
	}
	subs.Append(Subscriber{peerId, requestId, request})
	subs.RemoveContentFilters(peerId, requestId, request.ContentFilters)

	sub := firstSubscriber(subs, contentTopic)
	assert.Nil(t, sub)
}

func TestRemovePartial(t *testing.T) {
	subs := NewSubscribers(10 * time.Second)
	peerId := createPeerId(t)
	requestId := "request_1"
	topic1 := "topic1"
	topic2 := "topic2"
	request := &pb.FilterRequest{
		Subscribe:      true,
		Topic:          TOPIC,
		ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: topic1}, {ContentTopic: topic2}},
	}
	subs.Append(Subscriber{peerId, requestId, request})
	subs.RemoveContentFilters(peerId, requestId, []*pb.FilterRequest_ContentFilter{{ContentTopic: topic1}})

	sub := firstSubscriber(subs, topic2)
	assert.NotNil(t, sub)
	assert.Len(t, sub.filter.ContentFilters, 1)
}

func TestRemoveDuplicateSubscriptions(t *testing.T) {
	subs := NewSubscribers(10 * time.Second)
	peerId := createPeerId(t)
	topic := "topic"
	requestId1 := "request_1"
	requestId2 := "request_2"
	request1 := &pb.FilterRequest{
		Subscribe:      true,
		Topic:          TOPIC,
		ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: topic}},
	}
	request2 := &pb.FilterRequest{
		Subscribe:      true,
		Topic:          TOPIC,
		ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: topic}},
	}
	subs.Append(Subscriber{peerId, requestId1, request1})
	subs.Append(Subscriber{peerId, requestId2, request2})
	subs.RemoveContentFilters(peerId, requestId2, []*pb.FilterRequest_ContentFilter{{ContentTopic: topic}})
	subs.RemoveContentFilters(peerId, requestId1, []*pb.FilterRequest_ContentFilter{{ContentTopic: topic}})

	sub := firstSubscriber(subs, topic)
	assert.Nil(t, sub)
}

func TestRemoveDuplicateSubscriptionsPartial(t *testing.T) {
	subs := NewSubscribers(10 * time.Second)
	peerId := createPeerId(t)
	topic := "topic"
	requestId1 := "request_1"
	requestId2 := "request_2"
	request1 := &pb.FilterRequest{
		Subscribe:      true,
		Topic:          TOPIC,
		ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: topic}},
	}
	request2 := &pb.FilterRequest{
		Subscribe:      true,
		Topic:          TOPIC,
		ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: topic}},
	}
	subs.Append(Subscriber{peerId, requestId1, request1})
	subs.Append(Subscriber{peerId, requestId2, request2})
	subs.RemoveContentFilters(peerId, requestId1, []*pb.FilterRequest_ContentFilter{{ContentTopic: topic}})

	sub := firstSubscriber(subs, topic)
	assert.NotNil(t, sub)
	assert.Equal(t, sub.requestId, requestId2)
}

func TestRemoveBogus(t *testing.T) {
	subs := NewSubscribers(10 * time.Second)
	peerId := createPeerId(t)
	requestId := "request_1"
	contentTopic := "topic1"
	request := &pb.FilterRequest{
		Subscribe:      true,
		Topic:          TOPIC,
		ContentFilters: []*pb.FilterRequest_ContentFilter{{ContentTopic: contentTopic}},
	}
	subs.Append(Subscriber{peerId, requestId, request})
	subs.RemoveContentFilters(peerId, requestId, []*pb.FilterRequest_ContentFilter{{ContentTopic: "does not exist"}, {ContentTopic: contentTopic}})

	sub := firstSubscriber(subs, contentTopic)
	assert.Nil(t, sub)
}
