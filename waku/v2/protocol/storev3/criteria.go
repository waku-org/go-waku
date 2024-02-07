package storev3

import (
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/storev3/pb"
	"google.golang.org/protobuf/proto"
)

type Criteria interface {
	PopulateStoreRequest(request *pb.StoreRequest)
}

type FilterCriteria struct {
	protocol.ContentFilter
	TimeStart *int64
	TimeEnd   *int64
}

func (f FilterCriteria) PopulateStoreRequest(request *pb.StoreRequest) {
	request.ContentTopics = f.ContentTopicsList()
	request.PubsubTopic = proto.String(f.PubsubTopic)
	request.TimeStart = f.TimeStart
	request.TimeEnd = f.TimeEnd
}

type MessageHashCriteria struct {
	MessageHashes [][]byte
}

func (m MessageHashCriteria) PopulateStoreRequest(request *pb.StoreRequest) {
	request.MessageHashes = m.MessageHashes
}
