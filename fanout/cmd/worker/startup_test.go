package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Run the real entry point in a child process: bad credentials must fail before
// dialing Kafka or Redis. No real infrastructure is needed or started.
func TestStartupRejectsInvalidFanoutCredential(t *testing.T) {
	if os.Getenv("CHORDS_TEST_STARTUP") == "1" {
		if err := os.Chdir("../.."); err != nil {
			panic(err)
		}
		main()
		return
	}
	for _, token := range []string{"", "short", strings.Repeat("x", 32) + " "} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestStartupRejectsInvalidFanoutCredential$")
		for _, env := range os.Environ() {
			if !strings.HasPrefix(env, "FANOUT_TOKEN=") && !strings.HasPrefix(env, "CHORDS_TEST_STARTUP=") {
				cmd.Env = append(cmd.Env, env)
			}
		}
		cmd.Env = append(cmd.Env, "CHORDS_TEST_STARTUP=1", "FANOUT_TOKEN="+token)
		raw, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(raw), "FANOUT_TOKEN must") {
			t.Fatalf("startup did not fail on credential validation: %v %s", err, raw)
		}
		if strings.Contains(string(raw), "redis ping") || strings.Contains(string(raw), "kafka") {
			t.Fatalf("dependencies were attempted before credential validation: %s", raw)
		}
	}
}
