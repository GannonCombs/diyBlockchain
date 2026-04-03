package bcrypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/gannoncombs/diyBlockchain/core"
)

// SignTransaction signs a transaction with the given private key.
// It computes a hash of the transaction's content (excluding signature fields),
// signs that hash, and stores the signature + public key on the transaction.
func SignTransaction(tx *core.Transaction, privKey *ecdsa.PrivateKey) error {
	hash := TxSigningHash(*tx)

	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		return fmt.Errorf("signing transaction: %w", err)
	}

	// Encode signature as hex: r bytes + s bytes, each zero-padded to 32 bytes
	rBytes := padLeft(r.Bytes(), 32)
	sBytes := padLeft(s.Bytes(), 32)
	tx.Signature = hex.EncodeToString(append(rBytes, sBytes...))

	// Encode the public key so verifiers can recover the address
	pubBytes := elliptic.Marshal(privKey.PublicKey.Curve, privKey.PublicKey.X, privKey.PublicKey.Y)
	tx.PubKey = hex.EncodeToString(pubBytes)

	return nil
}

// VerifyTransaction checks that a transaction's signature is valid and
// that the signer's public key matches the From address.
func VerifyTransaction(tx core.Transaction) error {
	// Coinbase transactions are system-generated — no signature needed
	if tx.IsCoinbase() {
		return nil
	}

	if tx.Signature == "" || tx.PubKey == "" {
		return fmt.Errorf("transaction from %s is missing signature", tx.From)
	}

	// Decode public key
	pubBytes, err := hex.DecodeString(tx.PubKey)
	if err != nil {
		return fmt.Errorf("invalid public key encoding: %w", err)
	}

	x, y := elliptic.Unmarshal(elliptic.P256(), pubBytes)
	if x == nil {
		return fmt.Errorf("invalid public key")
	}

	pubKey := &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}

	// Verify the signer's address matches the From field
	derivedAddr := AddressFromPubKey(pubKey)
	if derivedAddr != tx.From {
		return fmt.Errorf("public key doesn't match sender address (got %s, expected %s)",
			derivedAddr, tx.From)
	}

	// Decode signature
	sigBytes, err := hex.DecodeString(tx.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}
	if len(sigBytes) != 64 {
		return fmt.Errorf("invalid signature length: expected 64, got %d", len(sigBytes))
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	// Verify the signature against the transaction hash
	hash := TxSigningHash(tx)
	if !ecdsa.Verify(pubKey, hash, r, s) {
		return fmt.Errorf("invalid signature for transaction from %s", tx.From)
	}

	return nil
}

// TxSigningHash computes the SHA-256 hash of the transaction's content fields.
// Signature and PubKey are excluded — they're not part of what gets signed.
func TxSigningHash(tx core.Transaction) []byte {
	data := fmt.Sprintf("%s%s%s%d%d", tx.Type, tx.From, tx.To, tx.Amount, tx.Timestamp)
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// padLeft zero-pads a byte slice to the given length.
func padLeft(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	padded := make([]byte, size)
	copy(padded[size-len(b):], b)
	return padded
}
