package dynamic

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestMetadata(t *testing.T) {

	metadata := &RLNMetadata{
		LastProcessedBlock: 128,
		ChainID:            big.NewInt(1155511),
		ContractAddress:    common.HexToAddress("0x9c09146844c1326c2dbc41c451766c7138f88155"),
	}

	serializedMetadata := metadata.Serialize()

	unserializedMetadata, err := DeserializeMetadata(serializedMetadata)
	require.NoError(t, err)
	require.Equal(t, metadata.ChainID.Uint64(), unserializedMetadata.ChainID.Uint64())
	require.Equal(t, metadata.LastProcessedBlock, unserializedMetadata.LastProcessedBlock)
	require.Equal(t, metadata.ContractAddress.Hex(), unserializedMetadata.ContractAddress.Hex())

	// Handle cases where the chainId is not specified (for some reason?)
	metadata.ChainID = nil
	serializedMetadata = metadata.Serialize()
	unserializedMetadata, err = DeserializeMetadata(serializedMetadata)
	require.NoError(t, err)
	require.Equal(t, uint64(0), unserializedMetadata.ChainID.Uint64())
}
