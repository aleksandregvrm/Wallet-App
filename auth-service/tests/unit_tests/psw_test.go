package unit_tests

import (
	"testing"

	"go-task-wallet-service/auth-service/internal/utils"
)

func TestHashPassword_EmptyPassword(t *testing.T) {
	hashed, err := utils.HashPassword("")
	if err == nil {
		t.Fatalf("expected an error for an empty password, got hash: %q", hashed)
	}
	if hashed != "" {
		t.Fatalf("expected empty hash on error, got: %q", hashed)
	}
}

func TestHashPassword_ProducesDifferentHashForSamePassword(t *testing.T) {
	hash1, err := utils.HashPassword("Hunter22!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hash2, err := utils.HashPassword("Hunter22!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash1 == hash2 {
		t.Fatalf("expected bcrypt salting to produce different hashes for the same password")
	}
}

func TestHashPassword_ThenComparePassword_Success(t *testing.T) {
	hashed, err := utils.HashPassword("Hunter22!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := utils.ComparePassword(hashed, "Hunter22!"); err != nil {
		t.Fatalf("expected the correct password to match, got: %v", err)
	}
}

func TestComparePassword_WrongPassword(t *testing.T) {
	hashed, err := utils.HashPassword("Hunter22!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := utils.ComparePassword(hashed, "WrongPassword1!"); err == nil {
		t.Fatalf("expected an error for a mismatched password")
	}
}

func TestComparePassword_MalformedHash(t *testing.T) {
	if err := utils.ComparePassword("not-a-real-bcrypt-hash", "Hunter22!"); err == nil {
		t.Fatalf("expected an error for a malformed hash")
	}
}
