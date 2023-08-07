package dynamic

import (
	"encoding/binary"
	"errors"
)

type RLNMetadata struct {
	LastProcessedBlock uint64
}

func (r RLNMetadata) Serialize() []byte {
	result := make([]byte, 8)
	binary.LittleEndian.PutUint64(result, r.LastProcessedBlock)
	return result
}

func DeserializeMetadata(b []byte) (RLNMetadata, error) {
	if len(b) != 8 {
		return RLNMetadata{}, errors.New("wrong size")
	}
	return RLNMetadata{
		LastProcessedBlock: binary.LittleEndian.Uint64(b),
	}, nil
}

func (gm *DynamicGroupManager) SetMetadata(meta RLNMetadata) error {
	b := meta.Serialize()
	return gm.rln.SetMetadata(b)
}

func (gm *DynamicGroupManager) GetMetadata() (RLNMetadata, error) {
	b, err := gm.rln.GetMetadata()
	if err != nil {
		return RLNMetadata{}, err
	}

	return DeserializeMetadata(b)
}
