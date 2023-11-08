package tests

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/crypto"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

// GetHostAddress returns the first listen address used by a host
func GetHostAddress(ha host.Host) multiaddr.Multiaddr {
	return ha.Addrs()[0]
}

// FindFreePort returns an available port number
func FindFreePort(t *testing.T, host string, maxAttempts int) (int, error) {
	t.Helper()

	if host == "" {
		host = "localhost"
	}

	for i := 0; i < maxAttempts; i++ {
		addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			t.Logf("unable to resolve tcp addr: %v", err)
			continue
		}
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			l.Close()
			t.Logf("unable to listen on addr %q: %v", addr, err)
			continue
		}

		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		return port, nil

	}

	return 0, fmt.Errorf("no free port found")
}

// FindFreePort returns an available port number
func FindFreeUDPPort(t *testing.T, host string, maxAttempts int) (int, error) {
	t.Helper()

	if host == "" {
		host = "localhost"
	}

	for i := 0; i < maxAttempts; i++ {
		addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, "0"))
		if err != nil {
			t.Logf("unable to resolve tcp addr: %v", err)
			continue
		}
		l, err := net.ListenUDP("udp", addr)
		if err != nil {
			l.Close()
			t.Logf("unable to listen on addr %q: %v", addr, err)
			continue
		}

		port := l.LocalAddr().(*net.UDPAddr).Port
		l.Close()
		return port, nil

	}

	return 0, fmt.Errorf("no free port found")
}

// MakeHost creates a Libp2p host with a random key on a specific port
func MakeHost(ctx context.Context, port int, randomness io.Reader) (host.Host, error) {
	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, randomness)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	ps, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, err
	}

	psWrapper := peerstore.NewWakuPeerstore(ps)
	if err != nil {
		return nil, err
	}

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	return libp2p.New(
		libp2p.Peerstore(psWrapper),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
}

// CreateWakuMessage creates a WakuMessage protobuffer with default values and a custom contenttopic and timestamp
func CreateWakuMessage(contentTopic string, timestamp *int64, optionalPayload ...string) *pb.WakuMessage {
	var payload []byte
	if len(optionalPayload) > 0 {
		payload = []byte(optionalPayload[0])
	} else {
		payload = []byte{1, 2, 3}
	}
	return &pb.WakuMessage{Payload: payload, ContentTopic: contentTopic, Timestamp: timestamp}
}

// RandomHex returns a random hex string of n bytes
func RandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func NewLocalnode(priv *ecdsa.PrivateKey, ipAddr *net.TCPAddr, udpPort int, wakuFlags wenr.WakuEnrBitfield, advertiseAddr *net.IP, log *zap.Logger) (*enode.LocalNode, error) {
	db, err := enode.OpenDB("")
	if err != nil {
		return nil, err
	}
	localnode := enode.NewLocalNode(db, priv)
	localnode.SetFallbackUDP(udpPort)
	localnode.Set(enr.WithEntry(wenr.WakuENRField, wakuFlags))
	localnode.SetFallbackIP(net.IP{127, 0, 0, 1})
	localnode.SetStaticIP(ipAddr.IP)

	if udpPort > 0 && udpPort <= math.MaxUint16 {
		localnode.Set(enr.UDP(uint16(udpPort))) // lgtm [go/incorrect-integer-conversion]
	} else {
		log.Error("setting udpPort", zap.Int("port", udpPort))
	}

	if ipAddr.Port > 0 && ipAddr.Port <= math.MaxUint16 {
		localnode.Set(enr.TCP(uint16(ipAddr.Port))) // lgtm [go/incorrect-integer-conversion]
	} else {
		log.Error("setting tcpPort", zap.Int("port", ipAddr.Port))
	}

	if advertiseAddr != nil {
		localnode.SetStaticIP(*advertiseAddr)
	}

	return localnode, nil
}

func CreateHost(t *testing.T, opts ...config.Option) (host.Host, int, *ecdsa.PrivateKey) {
	privKey, err := gcrypto.GenerateKey()
	require.NoError(t, err)

	sPrivKey := libp2pcrypto.PrivKey(utils.EcdsaPrivKeyToSecp256k1PrivKey(privKey))

	port, err := FindFreePort(t, "127.0.0.1", 3)
	require.NoError(t, err)

	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	require.NoError(t, err)

	opts = append(opts, libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(sPrivKey))

	host, err := libp2p.New(opts...)
	require.NoError(t, err)

	return host, port, privKey
}

func ExtractIP(addr multiaddr.Multiaddr) (*net.TCPAddr, error) {
	ipStr, err := addr.ValueForProtocol(multiaddr.P_IP4)
	if err != nil {
		return nil, err
	}

	portStr, err := addr.ValueForProtocol(multiaddr.P_TCP)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	return &net.TCPAddr{
		IP:   net.ParseIP(ipStr),
		Port: port,
	}, nil
}

func RandomInt(min, max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return 0, err
	}
	return min + int(n.Int64()), nil
}

func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)

	if err != nil {
		return nil, err
	}

	return b, nil
}

func GenerateRandomASCIIString(minLength int, maxLength int) (string, error) {
	length, err := rand.Int(rand.Reader, big.NewInt(int64(maxLength-minLength+1)))
	if err != nil {
		return "", err
	}
	length.SetInt64(length.Int64() + int64(minLength))

	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length.Int64())
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		result[i] = chars[num.Int64()]
	}

	return string(result), nil
}

func GenerateRandomUTF8String(minLength int, maxLength int) (string, error) {
	length, err := rand.Int(rand.Reader, big.NewInt(int64(maxLength-minLength+1)))
	if err != nil {
		return "", err
	}
	length.SetInt64(length.Int64() + int64(minLength))

	runes := make([]rune, length.Int64())
	for i := range runes {
		// Define unicode range
		start := 0x0020 // Space character
		end := 0x007F   // Tilde (~)

		randNum, err := rand.Int(rand.Reader, big.NewInt(int64(end-start+1)))
		if err != nil {
			return "", err
		}
		char := rune(start + int(randNum.Int64()))
		if !utf8.ValidRune(char) {
			i-- // Skip invalid runes
			continue
		}
		runes[i] = char
	}

	return string(runes), nil
}

func GenerateRandomUncommonUTF8String(minLength int, maxLength int) (string, error) {
	length, err := rand.Int(rand.Reader, big.NewInt(int64(maxLength-minLength+1)))
	if err != nil {
		return "", err
	}
	length.SetInt64(length.Int64() + int64(minLength))

	runes := make([]rune, length.Int64())
	for i := range runes {
		// Define unicode range for uncommon or unprintable characters, the Private Use Area (E000–F8FF)
		start := 0xE000
		end := 0xF8FF

		// Generate a random position within our range
		randNum, err := rand.Int(rand.Reader, big.NewInt(int64(end-start+1)))
		if err != nil {
			return "", err
		}
		char := rune(start + int(randNum.Int64()))
		if !utf8.ValidRune(char) {
			i-- // Skip invalid runes
			continue
		}
		runes[i] = char
	}

	return string(runes), nil
}

func GenerateRandomJSONString() (string, error) {
	// With 5 key-value pairs
	m := make(map[string]interface{})
	for i := 0; i < 5; i++ {
		key, err := GenerateRandomASCIIString(1, 20)
		if err != nil {
			return "", err
		}
		value, err := GenerateRandomASCIIString(1, 4097)
		if err != nil {
			return "", err
		}

		m[key] = value
	}

	// Marshal the map into a JSON string
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(m)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GenerateRandomBase64String(length int) (string, error) {
	bytes, err := RandomBytes(length)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bytes), nil
}

func GenerateRandomURLEncodedString(length int) (string, error) {
	randomString, err := GenerateRandomASCIIString(1, 4097)
	if err != nil {
		return "", err
	}

	// URL-encode the random string
	return url.QueryEscape(randomString), nil
}

func GenerateRandomSQLInsert() (string, error) {
	// Random table name
	tableName, err := GenerateRandomASCIIString(1, 10)
	if err != nil {
		return "", err
	}

	// Random column names
	columnCount, err := RandomInt(3, 6)
	if err != nil {
		return "", err
	}
	columnNames := make([]string, columnCount)
	for i := 0; i < columnCount; i++ {
		columnName, err := GenerateRandomASCIIString(1, 20)
		if err != nil {
			return "", err
		}
		columnNames[i] = columnName
	}

	// Random values
	values := make([]string, columnCount)
	for i := 0; i < columnCount; i++ {
		value, err := GenerateRandomASCIIString(1, 100)
		if err != nil {
			return "", err
		}
		values[i] = "'" + value + "'"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		tableName,
		strings.Join(columnNames, ", "),
		strings.Join(values, ", "))

	return query, nil
}
