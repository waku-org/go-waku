package keystore

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

// MembershipContract contains information about a membership smart contract address and the chain in which it is deployed
type MembershipContract struct {
	ChainID string `json:"chainId"`
	Address string `json:"address"`
}

// Equals is used to compare MembershipContract
func (m MembershipContract) Equals(other MembershipContract) bool {
	return m.Address == other.Address && m.ChainID == other.ChainID
}

// MembershipGroup contains information about the index in which a credential is stored in the merkle tree and the contract associated to this credential
type MembershipGroup struct {
	MembershipContract MembershipContract  `json:"membershipContract"`
	TreeIndex          rln.MembershipIndex `json:"treeIndex"`
}

// Equals is used to compare MembershipGroup
func (m MembershipGroup) Equals(other MembershipGroup) bool {
	return m.MembershipContract.Equals(other.MembershipContract) && m.TreeIndex == other.TreeIndex
}

// MembershipCredentials contains all the information about an RLN Identity Credential and membership group it belongs to
type MembershipCredentials struct {
	IdentityCredential *rln.IdentityCredential `json:"identityCredential"`
	MembershipGroups   []MembershipGroup       `json:"membershipGroups"`
}

// Equals is used to compare MembershipCredentials
func (m MembershipCredentials) Equals(other MembershipCredentials) bool {
	if !rln.IdentityCredentialEquals(*m.IdentityCredential, *other.IdentityCredential) {
		return false
	}

	for _, x := range m.MembershipGroups {
		found := false
		for _, y := range other.MembershipGroups {
			if x.Equals(y) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// AppInfo is a helper structure that contains information about the application that uses these credentials
type AppInfo struct {
	Application   string `json:"application"`
	AppIdentifier string `json:"appIdentifier"`
	Version       string `json:"version"`
}

// AppKeystore represents the membership credentials to be used in RLN
type AppKeystore struct {
	Application   string                  `json:"application"`
	AppIdentifier string                  `json:"appIdentifier"`
	Credentials   []appKeystoreCredential `json:"credentials"`
	Version       string                  `json:"version"`

	path   string
	logger *zap.Logger
}

type appKeystoreCredential struct {
	Crypto keystore.CryptoJSON `json:"crypto"`
}

const defaultSeparator = "\n"
