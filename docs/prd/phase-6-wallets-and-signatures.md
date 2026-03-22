# Phase 6: Wallets & Digital Signatures

## Goal
Replace plain-text account names with cryptographic wallets. Transactions must be signed by the sender's private key.

## What You'll Learn
- Public/private key cryptography (asymmetric encryption)
- How wallets work — a wallet IS a key pair
- Digital signatures — proving you authorized a transaction without revealing your private key
- Why blockchain addresses are derived from public keys
- Go: `crypto/ecdsa`, `crypto/elliptic`, `encoding/hex`

## Background: Why Signatures Matter
Up to this point, anyone can create a transaction "from" any account — there's no proof of identity. Digital signatures fix this:
1. You generate a key pair (private key = secret, public key = your address)
2. To send tokens, you sign the transaction with your private key
3. Anyone can verify the signature using your public key
4. Nobody can forge your signature without your private key

## Requirements

### Wallet
- Generate ECDSA key pair (P-256 curve)
- Address = hex-encoded hash of public key
- Save/load wallet to encrypted file

### Transaction Signing
- Sender signs transaction hash with their private key
- Signature included in transaction struct
- All nodes verify signature before accepting transaction

### Validation Update
- Reject transactions with missing or invalid signatures
- Coinbase/reward transactions are exempt (signed by protocol)

## Deliverables
- `crypto/wallet.go` — Key generation, address derivation
- `crypto/signature.go` — Sign and verify transactions
- `crypto/wallet_test.go` — Tests
- Update `core/transaction.go` with signature field
- Update validation logic throughout

## Success Criteria
- Can generate wallets with unique addresses
- Signed transactions are accepted, unsigned/forged ones are rejected
- Wallet files can be saved and reloaded
