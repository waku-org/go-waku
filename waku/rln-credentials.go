//go:build gowaku_rln
// +build gowaku_rln

package waku

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-waku/waku/v2/node"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

const RLN_CREDENTIALS_FILENAME = "rlnCredentials.txt"

func writeRLNMembershipCredentialsToFile(keyPair *rln.MembershipKeyPair, idx rln.MembershipIndex, contractAddress common.Address, path string, passwd []byte, overwrite bool) error {
	if path == "" {
		return nil // we dont want to use a credentials file
	}

	if keyPair == nil {
		return nil // no credentials to store
	}

	path = filepath.Join(path, RLN_CREDENTIALS_FILENAME)

	if err := checkForFileExistence(path, overwrite); err != nil {
		return err
	}

	credentialsJSON, err := json.Marshal(node.MembershipCredentials{
		Keypair:  keyPair,
		Index:    idx,
		Contract: contractAddress,
	})

	fmt.Println(string(credentialsJSON))
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

func loadMembershipCredentialsFromFile(credentialsFilePath string, passwd string) (node.MembershipCredentials, error) {
	src, err := ioutil.ReadFile(credentialsFilePath)
	if err != nil {
		return node.MembershipCredentials{}, err
	}

	var encryptedK keystore.CryptoJSON
	err = json.Unmarshal(src, &encryptedK)
	if err != nil {
		return node.MembershipCredentials{}, err
	}

	credentialsBytes, err := keystore.DecryptDataV3(encryptedK, passwd)
	if err != nil {
		return node.MembershipCredentials{}, err
	}

	var credentials node.MembershipCredentials
	err = json.Unmarshal(credentialsBytes, &credentials)

	return credentials, err
}

func getMembershipCredentials(logger *zap.Logger, options Options) (fromFile bool, credentials node.MembershipCredentials, err error) {
	if options.RLNRelay.CredentialsPath == "" { // Not using a file
		return false, node.MembershipCredentials{
			Contract: options.RLNRelay.MembershipContractAddress,
		}, nil
	}

	credentialsFilePath := filepath.Join(options.RLNRelay.CredentialsPath, RLN_CREDENTIALS_FILENAME)
	if _, err = os.Stat(credentialsFilePath); err == nil {
		if credentials, err := loadMembershipCredentialsFromFile(credentialsFilePath, options.KeyPasswd); err != nil {
			return false, node.MembershipCredentials{}, fmt.Errorf("could not read membership credentials file: %w", err)
		} else {
			logger.Info("loaded rln credentials", zap.String("filepath", credentialsFilePath))
			if (bytes.Equal(credentials.Contract.Bytes(), common.Address{}.Bytes())) {
				credentials.Contract = options.RLNRelay.MembershipContractAddress
			}
			return true, credentials, nil
		}
	}

	if os.IsNotExist(err) {
		return false, node.MembershipCredentials{
			Keypair:  nil,
			Index:    uint(options.RLNRelay.MembershipIndex),
			Contract: options.RLNRelay.MembershipContractAddress,
		}, nil

	}

	return false, node.MembershipCredentials{}, fmt.Errorf("could not read membership credentials file: %w", err)
}
