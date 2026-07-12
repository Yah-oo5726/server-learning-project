package auth

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestPassAuth(t *testing.T) {
	hash, err := HashPassword("password")
	if err != nil {
		t.Fatal(err)
	}
	equal, err := CheckPasswordHash("password", hash)
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Error("hash and password do not match")
	}
}

func TestFailAuth(t *testing.T) {
	hash, err := HashPassword("password")
	if err != nil {
		t.Fatal(err)
	}
	equal, err := CheckPasswordHash("wrongpassword", hash)
	if err != nil {
		t.Fatal(err)
	}
	if equal {
		t.Error("hash and password should not match")
	}
}

func TestJWTValid(t *testing.T) {
	userid := uuid.New()
	key, err := MakeJWT(userid, "secret", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	id, err := ValidateJWT(key, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if id != userid {
		t.Error("validated ID does not match expected ID")
	}
}

func TestJWTDifferentSecretKeys(t *testing.T) {
	userid := uuid.New()
	key, err := MakeJWT(userid, "secret", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	id, err := ValidateJWT(key, "different-secret")
	if err == nil {
		t.Error("expected error for different secret keys, got nil")
	}
	if id == userid {
		t.Error("returned ID should not match passed in ID")
	}
}

func TestJWTExpired(t *testing.T) {
	userid := uuid.New()
	key, err := MakeJWT(userid, "secret", time.Second*-1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ValidateJWT(key, "different-secret")
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}
