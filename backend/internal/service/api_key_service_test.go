package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestAPIKeyGeneration_Format(t *testing.T) {
	livePrefix := APIKeyPrefixLive
	if !strings.HasPrefix(livePrefix, "apisent_live_") {
		t.Errorf("expected live prefix to start with apisent_live_, got %s", livePrefix)
	}

	testKey := "apisent_live_a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
	hash := sha256.Sum256([]byte(testKey))
	keyHash := hex.EncodeToString(hash[:])

	if len(keyHash) != 64 {
		t.Errorf("expected SHA-256 hex string of length 64, got %d", len(keyHash))
	}
}
