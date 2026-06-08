package auth

import "testing"

func TestGenerateAndVerifyCodeHash(t *testing.T) {
	code, err := GenerateNumericCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6 digits, got %q", code)
	}
	hash := HashCode("login", "User@Example.com", code, "secret")
	if !VerifyCodeHash("login", "user@example.com", code, "secret", hash) {
		t.Fatal("expected code hash to verify")
	}
	if VerifyCodeHash("login", "user@example.com", "000000", "secret", hash) {
		t.Fatal("wrong code verified")
	}
}
