package persistence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRendezVousWithLevelDb(t *testing.T) {
	dbPath := "/tmp/rendezvoustest"
	rdv, err := NewRendezVousLevelDB(dbPath)
	require.NoError(t, err)

	err = rdv.Put([]byte("key"), []byte("value"))
	require.NoError(t, err)

	val, err := rdv.db.Get([]byte("key"), nil)
	require.NoError(t, err)
	require.Equal(t, []byte("value"), val)

	it := rdv.NewIterator(nil)
	ok := it.Next()
	require.True(t, ok)
	require.Equal(t, []byte("key"), it.Key())
	require.Equal(t, []byte("value"), it.Value())
	ok = it.Next()
	require.False(t, ok)

	err = rdv.Delete([]byte("key"))
	require.NoError(t, err)

	_, err = rdv.db.Get([]byte("key"), nil)
	require.Error(t, err)
}
