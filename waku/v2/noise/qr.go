package noise

import (
	"crypto/ed25519"
	b64 "encoding/base64"
	"errors"
	"strings"
)

type QR struct {
	applicationName    string
	applicationVersion string
	shardId            string
	ephemeralKey       ed25519.PublicKey
	committedStaticKey []byte
}

func NewQR(applicationName, applicationVersion, shardId string, ephemeralKey ed25519.PublicKey, committedStaticKey []byte) QR {
	return QR{
		applicationName:    applicationName,
		applicationVersion: applicationVersion,
		shardId:            shardId,
		ephemeralKey:       ephemeralKey,
		committedStaticKey: committedStaticKey,
	}
}

// Serializes input parameters to a base64 string for exposure through QR code (used by WakuPairing)
func (qr QR) String() string {
	return b64.StdEncoding.EncodeToString([]byte(qr.applicationName)) + ":" +
		b64.StdEncoding.EncodeToString([]byte(qr.applicationVersion)) + ":" +
		b64.StdEncoding.EncodeToString([]byte(qr.shardId)) + ":" +
		b64.StdEncoding.EncodeToString(qr.ephemeralKey) + ":" +
		b64.StdEncoding.EncodeToString(qr.committedStaticKey[:])
}

func (qr QR) Bytes() []byte {
	return []byte(qr.String())
}

// Deserializes input string in base64 to the corresponding (applicationName, applicationVersion, shardId, ephemeralKey, committedStaticKey)
func StringToQR(qrString string) (QR, error) {
	values := strings.Split(qrString, ":")
	if len(values) != 5 {
		return QR{}, errors.New("invalid qr string")
	}

	applicationName, err := b64.StdEncoding.DecodeString(values[0])
	if err != nil {
		return QR{}, err
	}

	applicationVersion, err := b64.StdEncoding.DecodeString(values[1])
	if err != nil {
		return QR{}, err
	}

	shardId, err := b64.StdEncoding.DecodeString(values[2])
	if err != nil {
		return QR{}, err
	}

	ephemeralKey, err := b64.StdEncoding.DecodeString(values[3])
	if err != nil {
		return QR{}, err
	}

	committedStaticKey, err := b64.StdEncoding.DecodeString(values[4])
	if err != nil {
		return QR{}, err
	}

	return QR{
		applicationName:    string(applicationName),
		applicationVersion: string(applicationVersion),
		shardId:            string(shardId),
		ephemeralKey:       ephemeralKey,
		committedStaticKey: committedStaticKey,
	}, nil
}
