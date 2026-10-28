package app

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthorizationEndpointConfiguration(t *testing.T) {
	t.Setenv("FANOUT_TOKEN", "isolated-fanout-test-credential-32-bytes")
	file := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(file, []byte("auth:\n  backend_url: http://configured:8080\n"), 0600))
	t.Setenv("CHORDS_AUTH_BACKEND_URL", "")
	cfg, err := LoadConfig(file)
	require.NoError(t, err)
	require.Equal(t, "http://configured:8080", cfg.Auth.BackendURL)
	t.Setenv("CHORDS_AUTH_BACKEND_URL", "http://backend:8080")
	cfg, err = LoadConfig(file)
	require.NoError(t, err)
	require.Equal(t, "http://backend:8080", cfg.Auth.BackendURL)
	t.Setenv("CHORDS_AUTH_BACKEND_URL", "file:///tmp/key")
	_, err = LoadConfig(file)
	require.Error(t, err)
}

func TestFanoutCredentialConfiguration(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(file, []byte("fanout:\n  token: yaml-must-not-enable-authentication\nredis:\n  password: private-redis-password\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"", "short", strings.Repeat("x", 31), strings.Repeat("x", 32) + " ", strings.Repeat("x", 32) + "\n", strings.Repeat("界", 32)} {
		t.Setenv("FANOUT_TOKEN", token)
		if _, err := LoadConfig(file); err == nil {
			t.Fatalf("invalid FANOUT_TOKEN was accepted")
		}
	}
	token := "isolated-fanout-test-credential-32-bytes"
	t.Setenv("FANOUT_TOKEN", token)
	output, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	original := os.Stdout
	os.Stdout = output
	defer func() { os.Stdout = original }()
	cfg, err := LoadConfig(file)
	os.Stdout = original
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Fanout.Token != token {
		t.Fatal("environment credential was not loaded")
	}
	raw, err := os.ReadFile(output.Name())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), token) || strings.Contains(string(raw), "private-redis-password") {
		t.Fatal("config log leaked credentials")
	}
}
