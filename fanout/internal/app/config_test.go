package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
