package filter

import (
	"context"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"strconv"
	"time"
)

func (s *FilterTestSuite) TestValidPayloadsASCII() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomASCIIString)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsUTF8() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomUTF8String)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsBase64() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomBase64String)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsJSON() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomJSONString)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsURLEncoded() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomURLEncodedString)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestValidPayloadsSQL() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare data
	messages := prepareData(100, false, false, true, tests.GenerateRandomSQLInsert)

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestLargePayloadsUTF8() {

	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 40*time.Second)

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Prepare basic data
	messages := prepareData(10, false, false, false, nil)

	// Generate large string
	for i := range messages {
		messages[i].payload, _ = tests.GenerateRandomUTF8String(153600)
		s.log.Info("Generated payload with ", zap.String("length", strconv.Itoa(len(messages[i].payload))))
	}

	// All messages should be received
	s.waitForMessages(func() {
		s.publishMessages(messages)
	}, s.subDetails, messages)

	_, err := s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestEmptyPayload() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Should get rejected
	_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch(), ""), relay.WithPubSubTopic(s.testTopic))
	s.Require().Error(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestEmptyContentTopic() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Should get rejected
	_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage("", utils.GetUnixEpoch(), "test_payload"), relay.WithPubSubTopic(s.testTopic))
	s.Require().Error(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestEmptyContentTopicEmptyPayload() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Should get rejected
	_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage("", utils.GetUnixEpoch(), ""), relay.WithPubSubTopic(s.testTopic))
	s.Require().Error(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

// Check message with non string payload is not received
// Check message with extra parameter is not received
// Check received message with extra option is received

//type WakuMessage struct {
//	state          protoimpl.MessageState
//	sizeCache      protoimpl.SizeCache
//	unknownFields  protoimpl.UnknownFields
//	Payload        []byte  `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
//	ContentTopic   string  `protobuf:"bytes,2,opt,name=content_topic,json=contentTopic,proto3" json:"content_topic,omitempty"`
//	Version        *uint32 `protobuf:"varint,3,opt,name=version,proto3,oneof" json:"version,omitempty"`
//	Timestamp      *int64  `protobuf:"zigzag64,10,opt,name=timestamp,proto3,oneof" json:"timestamp,omitempty"`
//	Meta           []byte  `protobuf:"bytes,11,opt,name=meta,proto3,oneof" json:"meta,omitempty"`
//	Ephemeral      *bool   `protobuf:"varint,31,opt,name=ephemeral,proto3,oneof" json:"ephemeral,omitempty"`
//	RateLimitProof []byte  `protobuf:"bytes,21,opt,name=rate_limit_proof,json=rateLimitProof,proto3,oneof" json:"rate_limit_proof,omitempty"`
//}

func (s *FilterTestSuite) TestTimestampInFuture() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	// Set time in the future
	timeDelta, _ := time.ParseDuration("200h")
	futureTime := proto.Int64(time.Now().UnixNano() + timeDelta.Nanoseconds())
	message := tests.CreateWakuMessage(s.testContentTopic, futureTime, "test_payload")

	// Should get accepted
	_, err := s.relayNode.Publish(s.ctx, message, relay.WithPubSubTopic(s.testTopic))
	s.Require().NoError(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestZeroTimestamp() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	message := tests.CreateWakuMessage(s.testContentTopic, new(int64), "test_payload")

	// Should get accepted
	_, err := s.relayNode.Publish(s.ctx, message, relay.WithPubSubTopic(s.testTopic))
	s.Require().NoError(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestNegativeTimestamp() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	negTimestamp := int64(-100)
	message := tests.CreateWakuMessage(s.testContentTopic, &negTimestamp, "test_payload")

	// Should get accepted
	_, err := s.relayNode.Publish(s.ctx, message, relay.WithPubSubTopic(s.testTopic))
	s.Require().NoError(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)

}

func (s *FilterTestSuite) TestFuturePayloadEncryptionVersion() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	message := tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch(), "test_payload")
	futureVersion := uint32(100)
	message.Version = &futureVersion

	// Should get accepted
	_, err := s.relayNode.Publish(s.ctx, message, relay.WithPubSubTopic(s.testTopic))
	s.Require().NoError(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
}

func (s *FilterTestSuite) TestMetaSizeOverLimit() {

	// Subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	message := tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch(), "test_payload")
	message.Meta = make([]byte, 65)

	// Should get accepted
	_, err := s.relayNode.Publish(s.ctx, message, relay.WithPubSubTopic(s.testTopic))
	s.Require().NoError(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
}
