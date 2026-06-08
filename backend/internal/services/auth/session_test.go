package auth

import "testing"

func TestSessionTokenHash(t *testing.T) {
	token, err := GenerateSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	hash := HashSessionToken(token, "secret")
	if hash == token {
		t.Fatal("hash must not equal plaintext token")
	}
	if HashSessionToken(token, "secret") != hash {
		t.Fatal("hash must be deterministic")
	}
	if HashSessionToken(token, "other") == hash {
		t.Fatal("secret must affect hash")
	}
}
