package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestConfigFile はテスト用のvenues.jsonファイルを作成します
func setupTestConfigFile(t *testing.T, content string) func() {
	t.Helper()

	// assetsディレクトリがなければ作成
	if err := os.MkdirAll("assets", 0755); err != nil {
		t.Fatalf("failed to create assets dir: %v", err)
	}

	// venuesConfigPath を一時的なものに差し替える
	originalPath := venuesConfigPath
	tmpPath := filepath.Join("assets", "test_venues.json")
	venuesConfigPath = tmpPath

	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	// クリーンアップ関数
	return func() {
		venuesConfigPath = originalPath
		os.Remove(tmpPath)
	}
}

func TestLoad_Success_FromFile(t *testing.T) {
	jsonContent := `[{"name":"ICLR","venue":"ICLR.cc/2025/Conference","year":2025}]`
	cleanup := setupTestConfigFile(t, jsonContent)
	defer cleanup()

	os.Setenv("TARGET_PLATFORM", "slack")
	os.Setenv("SLACK_BOT_TOKEN", "test_token")
	os.Setenv("SLACK_CHANNEL_ID", "test_channel")
	defer os.Unsetenv("TARGET_PLATFORM")
	defer os.Unsetenv("SLACK_BOT_TOKEN")
	defer os.Unsetenv("SLACK_CHANNEL_ID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(cfg.Venues) != 1 {
		t.Fatalf("expected 1 venue, got %d", len(cfg.Venues))
	}
	if cfg.Venues[0].Name != "ICLR" {
		t.Errorf("expected venue name 'ICLR', got '%s'", cfg.Venues[0].Name)
	}
}

func TestLoad_Failure_FileError(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		originalPath := venuesConfigPath
		venuesConfigPath = "assets/non_existent_venues.json"
		defer func() { venuesConfigPath = originalPath }()

		_, err := Load()
		if err == nil {
			t.Error("Load() should have failed for missing file, but it did not")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		cleanup := setupTestConfigFile(t, `[{"name":"ICLR"`) // 壊れたJSON
		defer cleanup()
		os.Setenv("TARGET_PLATFORM", "slack")
		defer os.Unsetenv("TARGET_PLATFORM")

		_, err := Load()
		if err == nil {
			t.Error("Load() should have failed for invalid JSON, but it did not")
		}
	})
}

func TestLoad_WithOpenReviewCredentials(t *testing.T) {
	t.Run("both credentials set", func(t *testing.T) {
		jsonContent := `[{"name":"ICLR","venue":"ICLR.cc/2025/Conference","year":2025}]`
		cleanup := setupTestConfigFile(t, jsonContent)
		defer cleanup()

		os.Setenv("TARGET_PLATFORM", "slack")
		os.Setenv("SLACK_BOT_TOKEN", "test_token")
		os.Setenv("SLACK_CHANNEL_ID", "test_channel")
		os.Setenv("OR_EMAIL", "user@example.com")
		os.Setenv("OR_PASSWORD", "secret")
		defer os.Unsetenv("TARGET_PLATFORM")
		defer os.Unsetenv("SLACK_BOT_TOKEN")
		defer os.Unsetenv("SLACK_CHANNEL_ID")
		defer os.Unsetenv("OR_EMAIL")
		defer os.Unsetenv("OR_PASSWORD")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}
		if cfg.OpenReviewEmail != "user@example.com" {
			t.Errorf("expected OpenReviewEmail 'user@example.com', got '%s'", cfg.OpenReviewEmail)
		}
		if cfg.OpenReviewPassword != "secret" {
			t.Errorf("expected OpenReviewPassword 'secret', got '%s'", cfg.OpenReviewPassword)
		}
	})

	t.Run("credentials not set", func(t *testing.T) {
		jsonContent := `[{"name":"ICLR","venue":"ICLR.cc/2025/Conference","year":2025}]`
		cleanup := setupTestConfigFile(t, jsonContent)
		defer cleanup()

		os.Setenv("TARGET_PLATFORM", "slack")
		os.Setenv("SLACK_BOT_TOKEN", "test_token")
		os.Setenv("SLACK_CHANNEL_ID", "test_channel")
		os.Unsetenv("OR_EMAIL")
		os.Unsetenv("OR_PASSWORD")
		defer os.Unsetenv("TARGET_PLATFORM")
		defer os.Unsetenv("SLACK_BOT_TOKEN")
		defer os.Unsetenv("SLACK_CHANNEL_ID")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}
		if cfg.OpenReviewEmail != "" {
			t.Errorf("expected empty OpenReviewEmail, got '%s'", cfg.OpenReviewEmail)
		}
		if cfg.OpenReviewPassword != "" {
			t.Errorf("expected empty OpenReviewPassword, got '%s'", cfg.OpenReviewPassword)
		}
	})
}
