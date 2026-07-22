package blob

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestR2ConfigFromEnv(t *testing.T) {
	for _, key := range []string{
		"WHISPER_R2_ACCOUNT_ID", "WHISPER_R2_ACCESS_KEY_ID",
		"WHISPER_R2_SECRET_ACCESS_KEY", "WHISPER_R2_BUCKET",
	} {
		t.Setenv(key, "")
	}
	if _, enabled, err := R2ConfigFromEnv(); err != nil || enabled {
		t.Fatalf("empty config = enabled %v, err %v", enabled, err)
	}
	t.Setenv("WHISPER_R2_ACCOUNT_ID", "account")
	if _, _, err := R2ConfigFromEnv(); err == nil {
		t.Fatal("partial config was accepted")
	}
	t.Setenv("WHISPER_R2_ACCESS_KEY_ID", "access")
	t.Setenv("WHISPER_R2_SECRET_ACCESS_KEY", "secret")
	t.Setenv("WHISPER_R2_BUCKET", "whisper-files")
	config, enabled, err := R2ConfigFromEnv()
	if err != nil || !enabled || config.Bucket != "whisper-files" {
		t.Fatalf("complete config = %#v, enabled %v, err %v", config, enabled, err)
	}
}

func TestR2PresignedRequestsUsePrivateS3Endpoint(t *testing.T) {
	store, err := NewR2Store(context.Background(), R2Config{
		AccountID: "example-account", AccessKeyID: "access", SecretAccessKey: "secret", Bucket: "files",
	})
	if err != nil {
		t.Fatal(err)
	}
	put, err := store.PresignPut(context.Background(), "attachments/user/file", "text/plain", 12, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(put.URL, "https://example-account.r2.cloudflarestorage.com/files/attachments/user/file?") {
		t.Fatalf("put URL = %q", put.URL)
	}
	if put.Headers["Content-Type"] != "text/plain" {
		t.Fatalf("put headers = %#v", put.Headers)
	}
	get, err := store.PresignGet(context.Background(), "attachments/user/file", "attachment; filename=file.txt", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(get.URL, "response-content-disposition=") {
		t.Fatalf("get URL does not sign content disposition: %q", get.URL)
	}
}
