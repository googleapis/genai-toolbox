package taskcrypto

import (
	"os"
	"testing"
	"time"
)

func TestEncryptDecryptTaskID(t *testing.T) {
	payload := TaskTokenPayload{
		Source:    "bigquery-prod",
		Engine:    "bigquery",
		NativeID:  "bqux_job_12345678_abcdef",
		CreatedAt: time.Now().Unix(),
	}

	token, err := EncryptTaskID(payload)
	if err != nil {
		t.Fatalf("EncryptTaskID failed: %v", err)
	}

	decrypted, err := DecryptTaskID(token)
	if err != nil {
		t.Fatalf("DecryptTaskID failed: %v", err)
	}

	if decrypted.Source != payload.Source ||
		decrypted.Engine != payload.Engine ||
		decrypted.NativeID != payload.NativeID ||
		decrypted.CreatedAt != payload.CreatedAt {
		t.Errorf("Decrypted payload %v does not match original %v", decrypted, payload)
	}
}

func TestDecryptTaskID_InvalidBase64(t *testing.T) {
	_, err := DecryptTaskID("invalid_base64!")
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got: %v", err)
	}
}

func TestDecryptTaskID_TamperedToken(t *testing.T) {
	payload := TaskTokenPayload{
		Source:    "bigquery-prod",
		Engine:    "bigquery",
		NativeID:  "bqux_job_12345678_abcdef",
		CreatedAt: time.Now().Unix(),
	}

	token, err := EncryptTaskID(payload)
	if err != nil {
		t.Fatalf("EncryptTaskID failed: %v", err)
	}

	// Tamper with the base64 token
	tamperedToken := token[:len(token)-5] + "AAAAA"
	_, err = DecryptTaskID(tamperedToken)
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken for tampered token, got: %v", err)
	}
}

func TestGetEncryptionKey(t *testing.T) {
	// Test default
	os.Unsetenv("MCP_TOOLBOX_TASK_ENCRYPTION_KEY")
	key := getEncryptionKey()
	if string(key) != string(defaultKey) {
		t.Errorf("Expected default key, got %v", key)
	}

	// Test custom
	customKey := "12345678901234567890123456789012"
	os.Setenv("MCP_TOOLBOX_TASK_ENCRYPTION_KEY", customKey)
	defer os.Unsetenv("MCP_TOOLBOX_TASK_ENCRYPTION_KEY")

	key = getEncryptionKey()
	if string(key) != customKey {
		t.Errorf("Expected custom key, got %v", key)
	}
}
