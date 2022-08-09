package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-rln/rln"
)

type membershipCredentials struct {
	Keypair rln.MembershipKeyPair `json:"keypair"`
	Index   rln.MembershipIndex   `json:"index"`
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return false

	} else if errors.Is(err, os.ErrNotExist) {
		return false

	} else {
		return false
	}
}

func writeRLNMembershipCredentialsToFile(path string, keyPair rln.MembershipKeyPair, idx rln.MembershipIndex) error {
	if fileExists(path) {
		return nil
	}

	credentialsJSON, err := json.Marshal(membershipCredentials{
		Keypair: keyPair,
		Index:   idx,
	})
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, credentialsJSON, 0600)
}

func loadMembershipCredentialsFromFile(path string) (rln.MembershipKeyPair, rln.MembershipIndex, error) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return rln.MembershipKeyPair{}, rln.MembershipIndex(0), err
	}

	var credentials membershipCredentials
	err = json.Unmarshal(src, &credentials)
	if err != nil {
		return rln.MembershipKeyPair{}, rln.MembershipIndex(0), err
	}

	return credentials.Keypair, credentials.Index, err
}

func getMembershipCredentials(path string, rlnIDKey string, rlnIDCommitment string, rlnMembershipIndex int) (idKey *rln.IDKey, idCommitment *rln.IDCommitment, index rln.MembershipIndex, err error) {
	if _, err = os.Stat(path); err == nil {
		if keyPair, index, err := loadMembershipCredentialsFromFile(path); err != nil {
			return nil, nil, rln.MembershipIndex(0), fmt.Errorf("could not read membership credentials file: %w", err)
		} else {
			return &keyPair.IDKey, &keyPair.IDCommitment, index, nil
		}
	}

	if os.IsNotExist(err) {
		if rlnIDKey != "" {
			idKey = new(rln.IDKey)
			copy((*idKey)[:], common.FromHex(rlnIDKey))
		}

		if rlnIDCommitment != "" {
			idCommitment = new(rln.IDCommitment)
			copy((*idCommitment)[:], common.FromHex(rlnIDCommitment))
		}

		return idKey, idCommitment, rln.MembershipIndex(rlnMembershipIndex), nil
	}

	return nil, nil, rln.MembershipIndex(0), fmt.Errorf("could not read membership credentials file: %w", err)
}
