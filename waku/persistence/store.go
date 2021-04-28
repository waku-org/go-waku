package persistence

import (
	"database/sql"
	"log"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
)

// DBStore is a MessageProvider that has a *sql.DB connection
type DBStore struct {
	store.MessageProvider
	db *sql.DB
}

type DBOption func(*DBStore) error

// WithDB is a DBOption that lets you use any custom *sql.DB with a DBStore.
func WithDB(db *sql.DB) DBOption {
	return func(d *DBStore) error {
		d.db = db
		return nil
	}
}

// WithDriver is a DBOption that will open a *sql.DB connection
func WithDriver(driverName string, datasourceName string) DBOption {
	return func(d *DBStore) error {
		db, err := sql.Open(driverName, datasourceName)
		if err != nil {
			return err
		}
		d.db = db
		return nil
	}
}

// Creates a new DB store using the db specified via options.
// It will create a messages table if it does not exist
func NewDBStore(opt DBOption) (*DBStore, error) {
	result := new(DBStore)

	err := opt(result)
	if err != nil {
		return nil, err
	}

	err = result.createTable()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (d *DBStore) createTable() error {
	sqlStmt := `CREATE TABLE IF NOT EXISTS message (
		id BLOB PRIMARY KEY,
		timestamp INTEGER NOT NULL,
		contentTopic BLOB NOT NULL,
		pubsubTopic BLOB NOT NULL,
		payload BLOB,
		version INTEGER NOT NULL DEFAULT 0
	) WITHOUT ROWID;`
	_, err := d.db.Exec(sqlStmt)
	if err != nil {
		return err
	}
	return nil
}

// Closes a DB connection
func (d *DBStore) Stop() {
	d.db.Close()
}

// Inserts a WakuMessage into the DB
func (d *DBStore) Put(cursor *pb.Index, pubsubTopic string, message *pb.WakuMessage) error {
	stmt, err := d.db.Prepare("INSERT INTO message (id, timestamp, contentTopic, pubsubTopic, payload, version) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(cursor.Digest, uint64(message.Timestamp), message.ContentTopic, pubsubTopic, message.Payload, message.Version)
	if err != nil {
		return err
	}

	return nil
}

// Returns all the stored WakuMessages
func (d *DBStore) GetAll() ([]*protocol.Envelope, error) {
	rows, err := d.db.Query("SELECT timestamp, contentTopic, pubsubTopic, payload, version FROM message ORDER BY timestamp ASC")
	if err != nil {
		return nil, err
	}

	var result []*protocol.Envelope

	defer rows.Close()

	for rows.Next() {
		var timestamp int64
		var contentTopic string
		var payload []byte
		var version uint32
		var pubsubTopic string

		err = rows.Scan(&timestamp, &contentTopic, &pubsubTopic, &payload, &version)
		if err != nil {
			log.Fatal(err)
		}

		msg := new(pb.WakuMessage)
		msg.ContentTopic = contentTopic
		msg.Payload = payload
		msg.Timestamp = float64(timestamp)
		msg.Version = version

		data, _ := msg.Marshal()
		envelope := protocol.NewEnvelope(msg, pubsubTopic, len(data), gcrypto.Keccak256(data))
		result = append(result, envelope)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}
