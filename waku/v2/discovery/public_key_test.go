package discovery

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"github.com/minio/sha256-simd"
	"github.com/stretchr/testify/require"
)

func TestBasicEquals(t *testing.T) {
	_, pub1, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	require.NoError(t, err)

	_, pub2, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	require.NoError(t, err)

	require.False(t, basicEquals(pub1, pub2))
	require.True(t, basicEquals(pub1, pub1))
}

func TestSignAndVerify(t *testing.T) {
	priv1, err := ecdsa.GenerateKey(crypto.ECDSACurve, rand.Reader)
	require.NoError(t, err)
	pub1 := ECDSAPublicKey{pub: &priv1.PublicKey}

	require.Equal(t, pb.KeyType_Secp256k1, pub1.Type())

	msg := []byte("hello world")

	data := sha256.Sum256(msg)
	sig, err := priv1.Sign(rand.Reader, data[:], nil)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := pub1.Verify(msg, sig)
	require.NoError(t, err)
	require.True(t, ok)

	// change data
	data[0] = ^data[0]
	ok, err = pub1.Verify(data[:], sig)
	require.NoError(t, err)
	require.False(t, ok)
}
