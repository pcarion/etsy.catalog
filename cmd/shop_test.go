package cmd

import "testing"

func TestUserIDFromToken(t *testing.T) {
	got, err := userIDFromToken("12345678.token-value")
	if err != nil {
		t.Fatal(err)
	}
	if got != "12345678" {
		t.Fatalf("user ID = %q", got)
	}
}

func TestUserIDFromTokenRejectsInvalidToken(t *testing.T) {
	for _, token := range []string{"", "token", "abc.token"} {
		if _, err := userIDFromToken(token); err == nil {
			t.Fatalf("expected %q to be rejected", token)
		}
	}
}
