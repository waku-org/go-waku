package keystore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

// DefaultCredentialsFilename is the default filename for the rln credentials keystore
const DefaultCredentialsFilename = "rlnKeystore.json"

// DefaultCredentialsPassword contains the default password used when no password is specified
const DefaultCredentialsPassword = "password"

// New creates a new instance of a rln credentials keystore
func New(keystorePath string, appInfo AppInfo, logger *zap.Logger) (*AppKeystore, error) {
	logger = logger.Named("rln-keystore")

	path := keystorePath
	if path == "" {
		logger.Warn("keystore: no credentials path set, using default path", zap.String("path", DefaultCredentialsFilename))
		path = DefaultCredentialsFilename
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// If no keystore exists at path we create a new empty one with passed keystore parameters
			err = createAppKeystore(path, appInfo, defaultSeparator)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	for _, keystoreBytes := range bytes.Split(src, []byte(defaultSeparator)) {
		if len(keystoreBytes) == 0 {
			continue
		}

		keystore := new(AppKeystore)
		keystore.logger = logger
		keystore.path = path
		err := json.Unmarshal(keystoreBytes, keystore)
		if err != nil {
			continue
		}

		if keystore.AppIdentifier == appInfo.AppIdentifier && keystore.Application == appInfo.Application && keystore.Version == appInfo.Version {
			return keystore, nil
		}
	}

	return nil, errors.New("no keystore found")
}

// GetMembershipCredentials decrypts and retrieves membership credentials from the keystore applying filters
func (k *AppKeystore) GetMembershipCredentials(keystorePassword string, filterIdentityCredentials []MembershipCredentials, filterMembershipContracts []MembershipContract) ([]MembershipCredentials, error) {
	password := keystorePassword
	if password == "" {
		k.logger.Warn("keystore: no credentials password set, using default password", zap.String("password", DefaultCredentialsPassword))
		password = DefaultCredentialsPassword
	}

	var result []MembershipCredentials

	for _, credential := range k.Credentials {
		credentialsBytes, err := keystore.DecryptDataV3(credential.Crypto, password)
		if err != nil {
			return nil, err
		}

		var credentials MembershipCredentials
		err = json.Unmarshal(credentialsBytes, &credentials)
		if err != nil {
			return nil, err
		}

		filteredCredential := filterCredential(credentials, filterIdentityCredentials, filterMembershipContracts)
		if filteredCredential != nil {
			result = append(result, *filteredCredential)
		}
	}

	return result, nil
}

// AddMembershipCredentials inserts a membership credential to the keystore matching the application, appIdentifier and version filters.
func (k *AppKeystore) AddMembershipCredentials(newIdentityCredential *rln.IdentityCredential, newMembershipGroup MembershipGroup, password string) (membershipGroupIndex uint, err error) {
	// A flag to tell us if the keystore contains a credential associated to the input identity credential, i.e. membershipCredential
	found := false
	for i, existingCredentials := range k.Credentials {
		credentialsBytes, err := keystore.DecryptDataV3(existingCredentials.Crypto, password)
		if err != nil {
			continue
		}

		var credentials MembershipCredentials
		err = json.Unmarshal(credentialsBytes, &credentials)
		if err != nil {
			continue
		}

		if rln.IdentityCredentialEquals(*credentials.IdentityCredential, *newIdentityCredential) {
			// idCredential is present in keystore. We add the input credential membership group to the one contained in the decrypted keystore credential (we deduplicate groups using sets)
			allMembershipsMap := make(map[MembershipGroup]struct{})
			for _, m := range credentials.MembershipGroups {
				allMembershipsMap[m] = struct{}{}
			}
			allMembershipsMap[newMembershipGroup] = struct{}{}

			// We sort membership groups, otherwise we will not have deterministic results in tests
			var allMemberships []MembershipGroup
			for k := range allMembershipsMap {
				allMemberships = append(allMemberships, k)
			}
			sort.Slice(allMemberships, func(i, j int) bool {
				return allMemberships[i].MembershipContract.Address < allMemberships[j].MembershipContract.Address
			})

			// we define the updated credential with the updated membership sets
			updatedCredential := MembershipCredentials{
				IdentityCredential: newIdentityCredential,
				MembershipGroups:   allMemberships,
			}

			// we re-encrypt creating a new keyfile
			b, err := json.Marshal(updatedCredential)
			if err != nil {
				return 0, err
			}

			encryptedCredentials, err := keystore.EncryptDataV3(b, []byte(password), keystore.StandardScryptN, keystore.StandardScryptP)
			if err != nil {
				return 0, err
			}

			// we update the original credential field in keystoreCredentials
			k.Credentials[i] = appKeystoreCredential{Crypto: encryptedCredentials}

			found = true

			// We setup the return values
			membershipGroupIndex = uint(len(allMemberships))
			for mIdx, mg := range updatedCredential.MembershipGroups {
				if mg.MembershipContract.Equals(newMembershipGroup.MembershipContract) {
					membershipGroupIndex = uint(mIdx)
					break
				}
			}

			// We stop decrypting other credentials in the keystore
			break
		}
	}

	if !found { // Not found
		newCredential := MembershipCredentials{
			IdentityCredential: newIdentityCredential,
			MembershipGroups:   []MembershipGroup{newMembershipGroup},
		}

		b, err := json.Marshal(newCredential)
		if err != nil {
			return 0, err
		}

		encryptedCredentials, err := keystore.EncryptDataV3(b, []byte(password), keystore.StandardScryptN, keystore.StandardScryptP)
		if err != nil {
			return 0, err
		}

		k.Credentials = append(k.Credentials, appKeystoreCredential{Crypto: encryptedCredentials})

		membershipGroupIndex = uint(len(newCredential.MembershipGroups) - 1)
	}

	return membershipGroupIndex, save(k, k.path)
}

func createAppKeystore(path string, appInfo AppInfo, separator string) error {
	if separator == "" {
		separator = defaultSeparator
	}

	keystore := AppKeystore{
		Application:   appInfo.Application,
		AppIdentifier: appInfo.AppIdentifier,
		Version:       appInfo.Version,
	}

	b, err := json.Marshal(keystore)
	if err != nil {
		return err
	}

	b = append(b, []byte(separator)...)

	buffer := new(bytes.Buffer)

	err = json.Compact(buffer, b)
	if err != nil {
		return err
	}

	return os.WriteFile(path, buffer.Bytes(), 0600)
}

func filterCredential(credential MembershipCredentials, filterIdentityCredentials []MembershipCredentials, filterMembershipContracts []MembershipContract) *MembershipCredentials {
	if len(filterIdentityCredentials) != 0 {
		found := false
		for _, filterCreds := range filterIdentityCredentials {
			if filterCreds.Equals(credential) {
				found = true
			}
		}
		if !found {
			return nil
		}
	}

	if len(filterMembershipContracts) != 0 {
		var membershipGroupsIntersection []MembershipGroup
		for _, filterContract := range filterMembershipContracts {
			for _, credentialGroups := range credential.MembershipGroups {
				if filterContract.Equals(credentialGroups.MembershipContract) {
					membershipGroupsIntersection = append(membershipGroupsIntersection, credentialGroups)
				}
			}
		}
		if len(membershipGroupsIntersection) != 0 {
			// If we have a match on some groups, we return the credential with filtered groups
			return &MembershipCredentials{
				IdentityCredential: credential.IdentityCredential,
				MembershipGroups:   membershipGroupsIntersection,
			}
		} else {
			return nil
		}
	}

	// We hit this return only if
	// - filterIdentityCredentials.len() == 0 and filterMembershipContracts.len() == 0 (no filter)
	// - filterIdentityCredentials.len() != 0 and filterMembershipContracts.len() == 0 (filter only on identity credential)
	// Indeed, filterMembershipContracts.len() != 0 will have its exclusive return based on all values of membershipGroupsIntersection.len()
	return &credential
}

// Safely saves a Keystore's JsonNode to disk.
// If exists, the destination file is renamed with extension .bkp; the file is written at its destination and the .bkp file is removed if write is successful, otherwise is restored
func save(keystore *AppKeystore, path string) error {
	// We first backup the current keystore
	_, err := os.Stat(path)
	if err == nil {
		err := os.Rename(path, path+".bkp")
		if err != nil {
			return err
		}
	}

	b, err := json.Marshal(keystore)
	if err != nil {
		return err
	}

	b = append(b, []byte(defaultSeparator)...)

	buffer := new(bytes.Buffer)

	err = json.Compact(buffer, b)
	if err != nil {
		restoreErr := os.Rename(path, path+".bkp")
		if restoreErr != nil {
			return fmt.Errorf("could not restore backup file: %w", restoreErr)
		}
		return err
	}

	err = os.WriteFile(path, buffer.Bytes(), 0600)
	if err != nil {
		restoreErr := os.Rename(path, path+".bkp")
		if restoreErr != nil {
			return fmt.Errorf("could not restore backup file: %w", restoreErr)
		}
		return err
	}

	// The write went fine, so we can remove the backup keystore
	_, err = os.Stat(path + ".bkp")
	if err == nil {
		err := os.Remove(path + ".bkp")
		if err != nil {
			return err
		}
	}

	return nil
}
