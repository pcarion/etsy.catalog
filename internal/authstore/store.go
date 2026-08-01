package authstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Credentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func Path() (string, error) {
	if name := os.Getenv("ETSY_CREDENTIALS_FILE"); name != "" {
		return name, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(dir, "etsy.catalog", "credentials.json"), nil
}

func Load() (Credentials, error) {
	name, err := Path()
	if err != nil {
		return Credentials{}, err
	}
	b, err := os.ReadFile(name)
	if os.IsNotExist(err) {
		return Credentials{}, nil
	}
	if err != nil {
		return Credentials{}, fmt.Errorf("read credentials: %w", err)
	}
	var credentials Credentials
	if err := json.Unmarshal(b, &credentials); err != nil {
		return Credentials{}, fmt.Errorf("parse credentials %s: %w", name, err)
	}
	return credentials, nil
}

func Save(credentials Credentials) error {
	name, err := Path()
	if err != nil {
		return err
	}
	dir := filepath.Dir(name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}
	b, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	temp, err := os.CreateTemp(dir, ".credentials-*")
	if err != nil {
		return fmt.Errorf("create temporary credentials file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err = temp.Write(b); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tempName, name); err != nil {
		return fmt.Errorf("save credentials: %w", err)
	}
	return nil
}
