package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/status-im/go-zerokit-rln/rln"
)

const RLN_CREDENTIALS_FILENAME = "rlnCredentials.txt"

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return false

	} else if errors.Is(err, os.ErrNotExist) {
		return false

	} else {
		return false
	}
}

func writeRLNMembershipCredentialsToFile(path string, keyPair *rln.MembershipKeyPair, idx rln.MembershipIndex, contractAddress common.Address) error {
	if path == "" {
		return nil // No path to save file
	}

	path = filepath.Join(path, RLN_CREDENTIALS_FILENAME)
	if fileExists(path) {
		return nil
	}

	if keyPair == nil {
		return nil // No credentials to write
	}

	credentialsJSON, err := json.Marshal(node.MembershipCredentials{
		Keypair: &rln.MembershipKeyPair{
			IDKey:        keyPair.IDKey,
			IDCommitment: keyPair.IDCommitment,
		},
		Index:    idx,
		Contract: contractAddress,
	})
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, credentialsJSON, 0600)
}

func loadMembershipCredentialsFromFile(rlnCredentialsPath string) (node.MembershipCredentials, error) {
	src, err := ioutil.ReadFile(rlnCredentialsPath)
	if err != nil {
		return node.MembershipCredentials{}, err
	}

	var credentials node.MembershipCredentials
	err = json.Unmarshal(src, &credentials)

	return credentials, err
}

func getMembershipCredentials(options RLNRelayOptions) (credentials node.MembershipCredentials, err error) {
	valuesWereInput := false
	if options.IDKey != "" || options.IDCommitment != "" {
		valuesWereInput = true
	}

	path := options.CredentialsPath

	if path == "" {
		return node.MembershipCredentials{
			Contract: options.MembershipContractAddress,
		}, nil
	}

	var osErr error
	if !valuesWereInput {
		path = filepath.Join(path, RLN_CREDENTIALS_FILENAME)
		if _, osErr = os.Stat(path); osErr == nil {
			if credentials, err := loadMembershipCredentialsFromFile(path); err != nil {
				return node.MembershipCredentials{}, fmt.Errorf("could not read membership credentials file: %w", err)
			} else {
				if (bytes.Equal(credentials.Contract.Bytes(), common.Address{}.Bytes())) {
					credentials.Contract = options.MembershipContractAddress
				}
				return credentials, nil
			}
		}
	}

	var keypair *rln.MembershipKeyPair
	if valuesWereInput || os.IsNotExist(osErr) {
		if options.IDKey != "" && options.IDCommitment != "" {
			keypair = new(rln.MembershipKeyPair)
			copy((keypair.IDKey)[:], common.FromHex(options.IDKey))
			copy((keypair.IDCommitment)[:], common.FromHex(options.IDCommitment))
		}

		return node.MembershipCredentials{
			Keypair:  keypair,
			Index:    uint(options.MembershipIndex),
			Contract: options.MembershipContractAddress,
		}, nil
	}

	return node.MembershipCredentials{}, fmt.Errorf("could not read membership credentials file: %w", err)
}
