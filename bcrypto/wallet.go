package bcrypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

// Wallet holds an ECDSA key pair. The private key signs transactions;
// the public key (compressed into an address) identifies the account.
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	Address    string // Hex-encoded, derived from public key
}

// walletFile is the JSON format for saving wallets to disk.
type walletFile struct {
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"` // Hex-encoded
	PublicKeyX string `json:"public_key_x"`
	PublicKeyY string `json:"public_key_y"`
}

// NewWallet generates a fresh ECDSA key pair on the P-256 curve.
func NewWallet() (*Wallet, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating key pair: %w", err)
	}

	addr := AddressFromPubKey(&privKey.PublicKey)
	return &Wallet{PrivateKey: privKey, Address: addr}, nil
}

// AddressFromPubKey derives a short address from an ECDSA public key.
// It concatenates the X and Y coordinates, SHA-256 hashes them,
// and takes the first 20 bytes (40 hex characters).
func AddressFromPubKey(pub *ecdsa.PublicKey) string {
	pubBytes := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	hash := sha256.Sum256(pubBytes)
	return hex.EncodeToString(hash[:20])
}

// Save writes the wallet to a JSON file in the given directory.
// The filename is the address with a .json extension.
func (w *Wallet) Save(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating wallet dir: %w", err)
	}

	wf := walletFile{
		Address:    w.Address,
		PrivateKey: hex.EncodeToString(w.PrivateKey.D.Bytes()),
		PublicKeyX: hex.EncodeToString(w.PrivateKey.PublicKey.X.Bytes()),
		PublicKeyY: hex.EncodeToString(w.PrivateKey.PublicKey.Y.Bytes()),
	}

	data, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing wallet: %w", err)
	}

	path := filepath.Join(dir, w.Address+".json")
	return os.WriteFile(path, data, 0600) // 0600 = owner read/write only
}

// LoadWallet reads a wallet from a JSON file.
func LoadWallet(path string) (*Wallet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading wallet file: %w", err)
	}

	var wf walletFile
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("parsing wallet file: %w", err)
	}

	privBytes, err := hex.DecodeString(wf.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %w", err)
	}
	pubXBytes, err := hex.DecodeString(wf.PublicKeyX)
	if err != nil {
		return nil, fmt.Errorf("decoding public key X: %w", err)
	}
	pubYBytes, err := hex.DecodeString(wf.PublicKeyY)
	if err != nil {
		return nil, fmt.Errorf("decoding public key Y: %w", err)
	}

	curve := elliptic.P256()
	privKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(pubXBytes),
			Y:     new(big.Int).SetBytes(pubYBytes),
		},
		D: new(big.Int).SetBytes(privBytes),
	}

	return &Wallet{
		PrivateKey: privKey,
		Address:    wf.Address,
	}, nil
}

// ListWallets returns the addresses of all wallets saved in the given directory.
func ListWallets(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var addresses []string
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) == ".json" {
			addresses = append(addresses, name[:len(name)-5])
		}
	}
	return addresses, nil
}
