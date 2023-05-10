package cliutils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type ProtectedTopic struct {
	Topic   string
	Address common.Address
}

func (p ProtectedTopic) String() string {
	return fmt.Sprintf("%s:%s", p.Topic, p.Address.String())
}

type ProtectedTopicSlice struct {
	Values *[]ProtectedTopic
}

func (k *ProtectedTopicSlice) Set(value string) error {
	protectedTopicParts := strings.Split(value, ":")
	if len(protectedTopicParts) != 2 {
		return errors.New("expected topic_name:hex_encoded_public_key")
	}

	if !common.IsHexAddress(protectedTopicParts[1]) {
		return errors.New("invalid address format")
	}

	*k.Values = append(*k.Values, ProtectedTopic{
		Topic:   protectedTopicParts[0],
		Address: common.HexToAddress(protectedTopicParts[1]),
	})
	return nil
}

func (v *ProtectedTopicSlice) String() string {
	if v.Values == nil {
		return ""
	}
	var output []string
	for _, v := range *v.Values {
		output = append(output, v.String())
	}

	return strings.Join(output, ", ")
}
