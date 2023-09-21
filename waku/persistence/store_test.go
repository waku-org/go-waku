//go:build include_postgres_tests
// +build include_postgres_tests

package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/mattn/go-sqlite3" // Blank import to register the sqlite3 driver
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/persistence/migrate"
	postgresmigration "github.com/waku-org/go-waku/waku/persistence/postgres/migrations"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func Migrate(db *sql.DB) error {
	migrationDriver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: "gowaku_" + postgres.DefaultMigrationsTable,
	})
	if err != nil {
		return err
	}
	return migrate.Migrate(db, migrationDriver, postgresmigration.AssetNames(), postgresmigration.Asset)
}

func TestDbStore(t *testing.T) {
	db := NewMockPgDB()
	store, err := NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), WithDB(db), WithMigrations(Migrate))
	require.NoError(t, err)

	err = store.Start(context.Background(), timesource.NewDefaultClock())
	require.NoError(t, err)

	res, err := store.GetAll()
	require.NoError(t, err)
	require.Empty(t, res)

	err = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test", utils.GetUnixEpoch()), utils.GetUnixEpoch(), "test"))
	require.NoError(t, err)

	res, err = store.GetAll()
	require.NoError(t, err)
	require.NotEmpty(t, res)
}

func TestStoreRetention(t *testing.T) {
	db := NewMockPgDB()
	store, err := NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), WithDB(db), WithMigrations(Migrate), WithRetentionPolicy(5, 20*time.Second))
	require.NoError(t, err)

	err = store.Start(context.Background(), timesource.NewDefaultClock())
	require.NoError(t, err)

	insertTime := time.Now()
	//////////////////////////////////
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test1", insertTime.Add(-70*time.Second).UnixNano()), insertTime.Add(-70*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test2", insertTime.Add(-60*time.Second).UnixNano()), insertTime.Add(-60*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test3", insertTime.Add(-50*time.Second).UnixNano()), insertTime.Add(-50*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test4", insertTime.Add(-40*time.Second).UnixNano()), insertTime.Add(-40*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test5", insertTime.Add(-30*time.Second).UnixNano()), insertTime.Add(-30*time.Second).UnixNano(), "test"))

	dbResults, err := store.GetAll()
	require.NoError(t, err)
	require.Len(t, dbResults, 5)

	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test6", insertTime.Add(-20*time.Second).UnixNano()), insertTime.Add(-20*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test7", insertTime.Add(-10*time.Second).UnixNano()), insertTime.Add(-10*time.Second).UnixNano(), "test"))

	// This step simulates starting go-waku again from scratch

	store, err = NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), WithDB(db), WithRetentionPolicy(5, 40*time.Second))
	require.NoError(t, err)

	err = store.Start(context.Background(), timesource.NewDefaultClock())
	require.NoError(t, err)

	dbResults, err = store.GetAll()
	require.NoError(t, err)
	require.Len(t, dbResults, 3)
	require.Equal(t, "test5", dbResults[0].Message.ContentTopic)
	require.Equal(t, "test6", dbResults[1].Message.ContentTopic)
	require.Equal(t, "test7", dbResults[2].Message.ContentTopic)
	// checking the number of all the message in the db
	msgCount, err := store.Count()
	require.NoError(t, err)
	require.Equal(t, msgCount, 3)
}

func TestQuery(t *testing.T) {
	db := NewMockPgDB()

	store, err := NewDBStore(prometheus.DefaultRegisterer, utils.Logger(), WithDB(db), WithMigrations(Migrate), WithRetentionPolicy(5, 20*time.Second))
	require.NoError(t, err)

	insertTime := time.Now()
	//////////////////////////////////
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test1", insertTime.Add(-40*time.Second).UnixNano()), insertTime.Add(-10*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test2", insertTime.Add(-30*time.Second).UnixNano()), insertTime.Add(-10*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test3", insertTime.Add(-20*time.Second).UnixNano()), insertTime.Add(-10*time.Second).UnixNano(), "test"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test3", insertTime.Add(-20*time.Second).UnixNano()), insertTime.Add(-10*time.Second).UnixNano(), "test2"))
	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test4", insertTime.Add(-10*time.Second).UnixNano()), insertTime.Add(-10*time.Second).UnixNano(), "test"))

	//  Range [startTime-endTime]
	// Check: matching ContentTopics and pubsubTopic, and ts of msg in range
	// this filters test1,test2 contentTopics. test3 matches list of contentTopic but its not within the time range
	cursor, msgs, err := store.Query(&pb.HistoryQuery{
		PubsubTopic: "test",
		ContentFilters: []*pb.ContentFilter{
			{ContentTopic: "test1"},
			{ContentTopic: "test2"},
			{ContentTopic: "test3"},
		},
		PagingInfo: &pb.PagingInfo{PageSize: 10},
		StartTime:  insertTime.Add(-40 * time.Second).UnixNano(),
		EndTime:    insertTime.Add(-20 * time.Second).UnixNano(),
	})
	require.NoError(t, err)
	require.Len(t, msgs, 3)

	_ = store.Put(protocol.NewEnvelope(tests.CreateWakuMessage("test5", insertTime.UnixNano()), insertTime.Add(-10*time.Second).UnixNano(), "test"))

	// Range [cursor-endTime]
	// Check:  matching ContentTopic,pubsubTopic, pageSize
	// cursor has last message id of test2 which is now-30second
	// endTime is now+1sec
	// matched messages are both test4,test5, but the len of returned result depends on pageSize
	var cursor2 *pb.Index
	for _, pageSize := range []int{1, 2} {
		cursorLocal, msgs, err := store.Query(&pb.HistoryQuery{
			PubsubTopic: "test",
			ContentFilters: []*pb.ContentFilter{
				{ContentTopic: "test4"},
				{ContentTopic: "test5"},
			},
			PagingInfo: &pb.PagingInfo{Cursor: cursor, PageSize: uint64(pageSize), Direction: pb.PagingInfo_FORWARD},
			EndTime:    insertTime.Add(1 * time.Second).UnixNano(),
		})
		require.NoError(t, err)
		require.Len(t, msgs, pageSize) // due to pageSize
		require.Equal(t, msgs[0].Message.ContentTopic, "test4")
		if len(msgs) > 1 {
			require.Equal(t, msgs[1].Message.ContentTopic, "test5")
		}
		cursor2 = cursorLocal
	}

	// range [startTime-cursor(test5_ContentTopic_Msg)],
	// Check: backend range with cursor excludes test1 ContentTopic,  matching ContentTopic
	// check backward pagination
	_, msgs, err = store.Query(&pb.HistoryQuery{
		PubsubTopic: "test",
		ContentFilters: []*pb.ContentFilter{
			{ContentTopic: "test1"}, // contentTopic test1 is out of the range
			{ContentTopic: "test2"},
			{ContentTopic: "test3"},
			{ContentTopic: "test4"},
		},
		PagingInfo: &pb.PagingInfo{Cursor: cursor2, PageSize: 4, Direction: pb.PagingInfo_BACKWARD},
		StartTime:  insertTime.Add(-39 * time.Second).UnixNano(),
	})
	require.NoError(t, err)
	require.Len(t, msgs, 3) // due to pageSize
	// Check:this also makes returned messages are sorted ascending
	for ind, msg := range msgs {
		require.Equal(t, msg.Message.ContentTopic, fmt.Sprintf("test%d", ind+2)) // test2,test3,test4
	}

	// checking most recent timestamp in db
	timestamp, err := store.MostRecentTimestamp()
	require.NoError(t, err)
	require.Equal(t, timestamp, insertTime.UnixNano())
}
