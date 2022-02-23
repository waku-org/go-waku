package store

import (
	"sort"
	"testing"

	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
	"github.com/stretchr/testify/require"
)

func TestIndexComputation(t *testing.T) {
	msg := &pb.WakuMessage{
		Payload:   []byte{1, 2, 3},
		Timestamp: utils.GetUnixEpoch(),
	}

	idx, err := computeIndex(protocol.NewEnvelope(msg, "test"))
	require.NoError(t, err)
	require.NotZero(t, idx.ReceiverTime)
	require.Equal(t, msg.Timestamp, idx.SenderTime)
	require.NotZero(t, idx.Digest)
	require.Len(t, idx.Digest, 32)

	msg1 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    123,
		ContentTopic: "/waku/2/default-content/proto",
	}
	idx1, err := computeIndex(protocol.NewEnvelope(msg1, "test"))
	require.NoError(t, err)

	msg2 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    123,
		ContentTopic: "/waku/2/default-content/proto",
	}
	idx2, err := computeIndex(protocol.NewEnvelope(msg2, "test"))
	require.NoError(t, err)

	require.Equal(t, idx1.Digest, idx2.Digest)
}

func TestIndexComparison(t *testing.T) {

	index1 := &pb.Index{
		ReceiverTime: 2,
		SenderTime:   1,
		Digest:       []byte{1},
	}

	index2 := &pb.Index{
		ReceiverTime: 2,
		SenderTime:   1,
		Digest:       []byte{2},
	}

	index3 := &pb.Index{
		ReceiverTime: 1,
		SenderTime:   2,
		Digest:       []byte{3},
	}

	iwm1 := IndexedWakuMessage{index: index1}
	iwm2 := IndexedWakuMessage{index: index2}
	iwm3 := IndexedWakuMessage{index: index3}

	require.Equal(t, 0, indexComparison(index1, index1))
	require.Equal(t, -1, indexComparison(index1, index2))
	require.Equal(t, 1, indexComparison(index2, index1))
	require.Equal(t, -1, indexComparison(index1, index3))
	require.Equal(t, 1, indexComparison(index3, index1))

	require.Equal(t, 0, indexedWakuMessageComparison(iwm1, iwm1))
	require.Equal(t, -1, indexedWakuMessageComparison(iwm1, iwm2))
	require.Equal(t, 1, indexedWakuMessageComparison(iwm2, iwm1))
	require.Equal(t, -1, indexedWakuMessageComparison(iwm1, iwm3))
	require.Equal(t, 1, indexedWakuMessageComparison(iwm3, iwm1))

	sortingList := []IndexedWakuMessage{iwm3, iwm1, iwm2}
	sort.Slice(sortingList, func(i, j int) bool {
		return indexedWakuMessageComparison(sortingList[i], sortingList[j]) == -1
	})

	require.Equal(t, iwm1, sortingList[0])
	require.Equal(t, iwm2, sortingList[1])
	require.Equal(t, iwm3, sortingList[2])
}

func createSampleList(s int) []IndexedWakuMessage {
	var result []IndexedWakuMessage
	for i := 0; i < s; i++ {
		result = append(result, IndexedWakuMessage{
			msg: &pb.WakuMessage{
				Payload: []byte{byte(i)},
			},
			index: &pb.Index{
				ReceiverTime: int64(i),
				SenderTime:   int64(i),
				Digest:       []byte{1},
			},
		})
	}
	return result
}

func TestFindIndex(t *testing.T) {
	msgList := createSampleList(10)
	require.Equal(t, 3, findIndex(msgList, msgList[3].index))
	require.Equal(t, -1, findIndex(msgList, &pb.Index{}))
}

func TestForwardPagination(t *testing.T) {
	msgList := createSampleList(10)

	// test for a normal pagination
	pagingInfo := &pb.PagingInfo{PageSize: 2, Cursor: msgList[3].index, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo := paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 2)
	require.Equal(t, []*pb.WakuMessage{msgList[4].msg, msgList[5].msg}, messages)
	require.Equal(t, msgList[5].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, pagingInfo.PageSize, newPagingInfo.PageSize)

	// test for an initial pagination request with an empty cursor
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 2)
	require.Equal(t, []*pb.WakuMessage{msgList[0].msg, msgList[1].msg}, messages)
	require.Equal(t, msgList[1].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, pagingInfo.PageSize, newPagingInfo.PageSize)

	// test for an initial pagination request with an empty cursor to fetch the entire history
	pagingInfo = &pb.PagingInfo{PageSize: 13, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 10)
	require.Equal(t, msgList[9].msg, messages[9])
	require.Equal(t, msgList[9].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(10), newPagingInfo.PageSize)

	// test for an empty msgList
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_FORWARD}
	var msgList2 []IndexedWakuMessage
	messages, newPagingInfo = paginateWithoutIndex(msgList2, pagingInfo)
	require.Len(t, messages, 0)
	require.Equal(t, pagingInfo.Cursor, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test for a page size larger than the remaining messages
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: msgList[3].index, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 6)
	require.Equal(t, []*pb.WakuMessage{msgList[4].msg, msgList[5].msg, msgList[6].msg, msgList[7].msg, msgList[8].msg, msgList[9].msg}, messages)
	require.Equal(t, msgList[9].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(6), newPagingInfo.PageSize)

	// test for a page size larger than the maximum allowed page size
	pagingInfo = &pb.PagingInfo{PageSize: MaxPageSize + 1, Cursor: msgList[3].index, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.True(t, len(messages) <= MaxPageSize)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.True(t, newPagingInfo.PageSize <= MaxPageSize)

	// test for a cursor pointing to the end of the message list
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: msgList[9].index, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 0)
	require.Equal(t, msgList[9].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test for an invalid cursor
	invalidIndex, err := computeIndex(protocol.NewEnvelope(&pb.WakuMessage{Payload: []byte{255, 255, 255}}, "test"))
	require.NoError(t, err)
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: invalidIndex, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 0)
	require.Equal(t, pagingInfo.Cursor, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test initial paging query over a message list with one message
	singleItemMsgList := msgList[0:1]
	pagingInfo = &pb.PagingInfo{PageSize: 10, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo = paginateWithoutIndex(singleItemMsgList, pagingInfo)
	require.Len(t, messages, 1)
	require.Equal(t, msgList[0].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(1), newPagingInfo.PageSize)
}

func TestBackwardPagination(t *testing.T) {
	msgList := createSampleList(10)

	// test for a normal pagination
	pagingInfo := &pb.PagingInfo{PageSize: 2, Cursor: msgList[3].index, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo := paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 2)
	require.Equal(t, []*pb.WakuMessage{msgList[1].msg, msgList[2].msg}, messages)
	require.Equal(t, msgList[1].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, pagingInfo.PageSize, newPagingInfo.PageSize)

	// test for an initial pagination request with an empty cursor
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 2)
	require.Equal(t, []*pb.WakuMessage{msgList[8].msg, msgList[9].msg}, messages)
	require.Equal(t, msgList[8].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, pagingInfo.PageSize, newPagingInfo.PageSize)

	// test for an initial pagination request with an empty cursor to fetch the entire history
	pagingInfo = &pb.PagingInfo{PageSize: 13, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 10)
	require.Equal(t, msgList[0].msg, messages[0])
	require.Equal(t, msgList[9].msg, messages[9])
	require.Equal(t, msgList[0].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(10), newPagingInfo.PageSize)

	// test for an empty msgList
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_BACKWARD}
	var msgList2 []IndexedWakuMessage
	messages, newPagingInfo = paginateWithoutIndex(msgList2, pagingInfo)
	require.Len(t, messages, 0)
	require.Equal(t, pagingInfo.Cursor, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test for a page size larger than the remaining messages
	pagingInfo = &pb.PagingInfo{PageSize: 5, Cursor: msgList[3].index, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 3)
	require.Equal(t, []*pb.WakuMessage{msgList[0].msg, msgList[1].msg, msgList[2].msg}, messages)
	require.Equal(t, msgList[0].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(3), newPagingInfo.PageSize)

	// test for a page size larger than the maximum allowed page size
	pagingInfo = &pb.PagingInfo{PageSize: MaxPageSize + 1, Cursor: msgList[3].index, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.True(t, len(messages) <= MaxPageSize)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.True(t, newPagingInfo.PageSize <= MaxPageSize)

	// test for a cursor pointing to the beginning of the message list
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: msgList[0].index, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 0)
	require.Equal(t, msgList[0].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test for an invalid cursor
	invalidIndex, err := computeIndex(protocol.NewEnvelope(&pb.WakuMessage{Payload: []byte{255, 255, 255}}, "test"))
	require.NoError(t, err)
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: invalidIndex, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo = paginateWithoutIndex(msgList, pagingInfo)
	require.Len(t, messages, 0)
	require.Equal(t, pagingInfo.Cursor, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test initial paging query over a message list with one message
	singleItemMsgList := msgList[0:1]
	pagingInfo = &pb.PagingInfo{PageSize: 10, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo = paginateWithoutIndex(singleItemMsgList, pagingInfo)
	require.Len(t, messages, 1)
	require.Equal(t, msgList[0].index, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(1), newPagingInfo.PageSize)
}
