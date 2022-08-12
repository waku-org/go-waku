//go:build gowaku_rln
// +build gowaku_rln

package waku

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-rln/rln"
)

type membershipCredentials struct {
	Keypair rln.MembershipKeyPair `json:"keypair"`
	Index   rln.MembershipIndex   `json:"index"`
}

func writeRLNMembershipCredentialsToFile(keyPair rln.MembershipKeyPair, idx rln.MembershipIndex, path string, passwd []byte, overwrite bool) error {
	if err := checkForFileExistence(path, overwrite); err != nil {
		return err
	}

	credentialsJSON, err := json.Marshal(membershipCredentials{
		Keypair: keyPair,
		Index:   idx,
	})
	if err != nil {
		return err
	}

	encryptedCredentials, err := keystore.EncryptDataV3(credentialsJSON, passwd, keystore.StandardScryptN, keystore.StandardScryptP)
	if err != nil {
		return err
	}

	output, err := json.Marshal(encryptedCredentials)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, output, 0600)
}

func loadMembershipCredentialsFromFile(path string, passwd string) (rln.MembershipKeyPair, rln.MembershipIndex, error) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return rln.MembershipKeyPair{}, rln.MembershipIndex(0), err
	}

	var encryptedK keystore.CryptoJSON
	err = json.Unmarshal(src, &encryptedK)
	if err != nil {
		return rln.MembershipKeyPair{}, rln.MembershipIndex(0), err
	}

	credentialsBytes, err := keystore.DecryptDataV3(encryptedK, passwd)
	if err != nil {
		return rln.MembershipKeyPair{}, rln.MembershipIndex(0), err
	}

	var credentials membershipCredentials
	err = json.Unmarshal(credentialsBytes, &credentials)
	if err != nil {
		return rln.MembershipKeyPair{}, rln.MembershipIndex(0), err
	}

	return credentials.Keypair, credentials.Index, err
}

func getMembershipCredentials(options Options) (fromFile bool, idKey *rln.IDKey, idCommitment *rln.IDCommitment, index rln.MembershipIndex, err error) {
	if _, err = os.Stat(options.RLNRelay.CredentialsFile); err == nil {
		if keyPair, index, err := loadMembershipCredentialsFromFile(options.RLNRelay.CredentialsFile, options.KeyPasswd); err != nil {
			return false, nil, nil, rln.MembershipIndex(0), fmt.Errorf("could not read membership credentials file: %w", err)
		} else {
			return true, &keyPair.IDKey, &keyPair.IDCommitment, index, nil
		}
	}

	if os.IsNotExist(err) {
		if options.RLNRelay.IDKey != "" {
			idKey = new(rln.IDKey)
			copy((*idKey)[:], common.FromHex(options.RLNRelay.IDKey))
		}

		if options.RLNRelay.IDCommitment != "" {
			idCommitment = new(rln.IDCommitment)
			copy((*idCommitment)[:], common.FromHex(options.RLNRelay.IDCommitment))
		}

		return false, idKey, idCommitment, rln.MembershipIndex(options.RLNRelay.MembershipIndex), nil

	}

	return false, nil, nil, rln.MembershipIndex(0), fmt.Errorf("could not read membership credentials file: %w", err)
}
