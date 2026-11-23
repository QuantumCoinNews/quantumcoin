package consensus

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

func TestNetworkSigningDigestV2RejectsReplayOfSamePayload(t *testing.T) {
	payload := sha256.Sum256([]byte("same transaction and outpoint"))
	left, err := NetworkSigningDigestV2(sha256.Sum256([]byte("testnet genesis")), payload)
	if err != nil {
		t.Fatal(err)
	}
	right, err := NetworkSigningDigestV2(sha256.Sum256([]byte("mainnet genesis")), payload)
	if err != nil {
		t.Fatal(err)
	}
	private, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := SignTransactionDigestV1(private, left)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := UncompressedPublicKeyV1(private)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyTransactionDigestSignatureV1(pub, left, signature) != nil ||
		VerifyTransactionDigestSignatureV1(pub, right, signature) == nil ||
		VerifyTransactionDigestSignatureV1(pub, payload, signature) == nil {
		t.Fatal("signature replay boundary was not enforced")
	}
	if _, err := NetworkSigningDigestV2([32]byte{}, payload); err == nil {
		t.Fatal("unbound network context accepted")
	}
}
