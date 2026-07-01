package auth

import (
	"testing"
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
