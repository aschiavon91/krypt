package krypt

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("DATABASE_URL=postgres://localhost/test\nAPI_KEY=secret123\n")

	ciphertext, err := encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	got, err := decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Errorf("got %q, want %q", got, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	wrongKey := make([]byte, 32)
	rand.Read(wrongKey)

	ciphertext, err := encrypt([]byte("secret"), key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = decrypt(ciphertext, wrongKey)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestDecryptCorruptedData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	ciphertext, err := encrypt([]byte("secret"), key)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt a byte in the middle
	ciphertext[len(ciphertext)/2] ^= 0xff

	_, err = decrypt(ciphertext, key)
	if err == nil {
		t.Fatal("expected error decrypting corrupted data")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	_, err := decrypt([]byte("short"), key)
	if err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

func TestEncryptUniqueNonce(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	plaintext := []byte("same content")

	c1, err := encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	c2, err := encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(c1, c2) {
		t.Error("two encryptions of same content produced identical ciphertext")
	}
}

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	if len(key) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(key))
	}

	// Verify it's valid hex by checking characters
	for _, c := range key {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("invalid hex char: %c", c)
		}
	}
}

func TestGenerateKeyUnique(t *testing.T) {
	k1, _ := GenerateKey()
	k2, _ := GenerateKey()

	if k1 == k2 {
		t.Error("two generated keys are identical")
	}
}
