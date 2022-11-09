package store

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestIndexComputation(t *testing.T) {
	msg := &pb.WakuMessage{
		Payload:   []byte{1, 2, 3},
		Timestamp: utils.GetUnixEpoch(),
	}

	idx := protocol.NewEnvelope(msg, utils.GetUnixEpoch(), "test").Index()
	require.NotZero(t, idx.ReceiverTime)
	require.Equal(t, msg.Timestamp, idx.SenderTime)
	require.NotZero(t, idx.Digest)
	require.Len(t, idx.Digest, 32)

	msg1 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    123,
		ContentTopic: "/waku/2/default-content/proto",
	}
	idx1 := protocol.NewEnvelope(msg1, utils.GetUnixEpoch(), "test").Index()

	msg2 := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    123,
		ContentTopic: "/waku/2/default-content/proto",
	}
	idx2 := protocol.NewEnvelope(msg2, utils.GetUnixEpoch(), "test").Index()

	require.Equal(t, idx1.Digest, idx2.Digest)
}

func createSampleList(s int) []*protocol.Envelope {
	var result []*protocol.Envelope
	for i := 0; i < s; i++ {
		msg :=
			&pb.WakuMessage{
				Payload:   []byte{byte(i)},
				Timestamp: int64(i),
			}
		result = append(result, protocol.NewEnvelope(msg, int64(i), "abc"))
	}
	return result
}

func TestForwardPagination(t *testing.T) {
	msgList := createSampleList(10)
	db := MemoryDB(t)
	for _, m := range msgList {
		err := db.Put(m)
		require.NoError(t, err)
	}

	// test for a normal pagination
	pagingInfo := &pb.PagingInfo{PageSize: 2, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err := findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, []*pb.WakuMessage{msgList[4].Message(), msgList[5].Message()}, messages)
	require.Equal(t, msgList[5].Index(), newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, pagingInfo.PageSize, newPagingInfo.PageSize)

	// test for an initial pagination request with an empty cursor
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, []*pb.WakuMessage{msgList[0].Message(), msgList[1].Message()}, messages)
	require.Equal(t, msgList[1].Index(), newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, pagingInfo.PageSize, newPagingInfo.PageSize)

	// test for an initial pagination request with an empty cursor to fetch the entire history
	pagingInfo = &pb.PagingInfo{PageSize: 13, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 10)
	require.Equal(t, msgList[9].Message(), messages[9])
	require.Equal(t, &pb.Index{}, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(10), newPagingInfo.PageSize)

	// test for an empty msgList
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, MemoryDB(t))
	require.NoError(t, err)
	require.Len(t, messages, 0)
	require.Equal(t, pagingInfo.Cursor, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test for a page size larger than the remaining messages
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 6)
	require.Equal(t, []*pb.WakuMessage{msgList[4].Message(), msgList[5].Message(), msgList[6].Message(), msgList[7].Message(), msgList[8].Message(), msgList[9].Message()}, messages)
	require.Equal(t, &pb.Index{}, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(6), newPagingInfo.PageSize)

	// test for a page size larger than the maximum allowed page size
	pagingInfo = &pb.PagingInfo{PageSize: MaxPageSize + 1, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.True(t, len(messages) <= MaxPageSize)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.True(t, newPagingInfo.PageSize <= MaxPageSize)

	// test for a cursor pointing to the end of the message list
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: msgList[9].Index(), Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 0)
	require.Equal(t, msgList[9].Index(), newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test for an invalid cursor
	invalidIndex := protocol.NewEnvelope(&pb.WakuMessage{Payload: []byte{255, 255, 255}}, utils.GetUnixEpoch(), "test").Index()
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: invalidIndex, Direction: pb.PagingInfo_FORWARD}
	_, _, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.ErrorIs(t, err, persistence.ErrInvalidCursor)
	require.Len(t, messages, 0)

	// test initial paging query over a message list with one message
	singleItemDB := MemoryDB(t)
	err = singleItemDB.Put(msgList[0])
	require.NoError(t, err)
	pagingInfo = &pb.PagingInfo{PageSize: 10, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, singleItemDB)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, &pb.Index{}, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(1), newPagingInfo.PageSize)
}

func TestBackwardPagination(t *testing.T) {
	msgList := createSampleList(10)
	db := MemoryDB(t)
	for _, m := range msgList {
		err := db.Put(m)
		require.NoError(t, err)
	}

	// test for a normal pagination
	pagingInfo := &pb.PagingInfo{PageSize: 2, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err := findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 2)

	require.Equal(t, []*pb.WakuMessage{msgList[1].Message(), msgList[2].Message()}, messages)
	require.Equal(t, msgList[1].Index(), newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, pagingInfo.PageSize, newPagingInfo.PageSize)

	// test for an initial pagination request with an empty cursor
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, []*pb.WakuMessage{msgList[8].Message(), msgList[9].Message()}, messages)
	require.Equal(t, msgList[8].Index(), newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, pagingInfo.PageSize, newPagingInfo.PageSize)

	// test for an initial pagination request with an empty cursor to fetch the entire history
	pagingInfo = &pb.PagingInfo{PageSize: 13, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 10)
	require.Equal(t, msgList[0].Message(), messages[0])
	require.Equal(t, msgList[9].Message(), messages[9])
	require.Equal(t, &pb.Index{}, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(10), newPagingInfo.PageSize)

	// test for an empty msgList
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, MemoryDB(t))
	require.NoError(t, err)
	require.Len(t, messages, 0)
	require.Equal(t, pagingInfo.Cursor, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test for a page size larger than the remaining messages
	pagingInfo = &pb.PagingInfo{PageSize: 5, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 3)
	require.Equal(t, []*pb.WakuMessage{msgList[0].Message(), msgList[1].Message(), msgList[2].Message()}, messages)
	require.Equal(t, &pb.Index{}, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(3), newPagingInfo.PageSize)

	// test for a page size larger than the maximum allowed page size
	pagingInfo = &pb.PagingInfo{PageSize: MaxPageSize + 1, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.True(t, len(messages) <= MaxPageSize)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.True(t, newPagingInfo.PageSize <= MaxPageSize)

	// test for a cursor pointing to the beginning of the message list
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: msgList[0].Index(), Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 0)
	require.Equal(t, msgList[0].Index(), newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test for an invalid cursor
	invalidIndex := protocol.NewEnvelope(&pb.WakuMessage{Payload: []byte{255, 255, 255}}, utils.GetUnixEpoch(), "test").Index()
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: invalidIndex, Direction: pb.PagingInfo_BACKWARD}
	_, _, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.ErrorIs(t, err, persistence.ErrInvalidCursor)
	require.Len(t, messages, 0)

	// test initial paging query over a message list with one message
	singleItemDB := MemoryDB(t)
	err = singleItemDB.Put(msgList[0])
	require.NoError(t, err)
	pagingInfo = &pb.PagingInfo{PageSize: 10, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, singleItemDB)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, &pb.Index{}, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(1), newPagingInfo.PageSize)
}
