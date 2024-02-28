package storev3

import (
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
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
	MessageHashes []wpb.MessageHash
}

func (m MessageHashCriteria) PopulateStoreRequest(request *pb.StoreRequest) {
	request.MessageHashes = make([][]byte, len(m.MessageHashes))
	for i := range m.MessageHashes {
		request.MessageHashes[i] = m.MessageHashes[i][:]
	}
}
