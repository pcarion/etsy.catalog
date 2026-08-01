package authstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	name := filepath.Join(t.TempDir(), "nested", "credentials.json")
	t.Setenv("ETSY_CREDENTIALS_FILE", name)
	want := Credentials{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Fatalf("got %#v", got)
	}
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("ETSY_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "missing.json"))
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "" {
		t.Fatalf("got %#v", got)
	}
}
