package service

import "os"

func init() { _ = os.Setenv("JWT_SECRET", "isolated-test-secret-for-chords-accounts") }
