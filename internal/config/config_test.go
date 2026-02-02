package config

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// 正常な設定で環境変数をセット
	os.Setenv("OR_VENUES_JSON", `[{"name":"ICLR","venue":"ICLR.cc/2025/Conference","year":2025,"venueURL":"https://iclr.cc"}]`)
	os.Setenv("TARGET_PLATFORM", "slack")
	os.Setenv("SLACK_BOT_TOKEN", "test_token")
	os.Setenv("SLACK_CHANNEL_ID", "test_channel")

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
	if cfg.TargetPlatform != "slack" {
		t.Errorf("expected target platform 'slack', got '%s'", cfg.TargetPlatform)
	}

	// 環境変数をクリーンアップ
	os.Unsetenv("OR_VENUES_JSON")
	os.Unsetenv("TARGET_PLATFORM")
	os.Unsetenv("SLACK_BOT_TOKEN")
	os.Unsetenv("SLACK_CHANNEL_ID")
}

func TestLoad_Failure(t *testing.T) {
	testCases := []struct {
		name    string
		envVars map[string]string
	}{
		{
			name:    "missing OR_VENUES_JSON",
			envVars: map[string]string{"TARGET_PLATFORM": "slack"},
		},
		{
			name:    "invalid OR_VENUES_JSON",
			envVars: map[string]string{"OR_VENUES_JSON": `[{"name":"ICLR"`, "TARGET_PLATFORM": "slack"},
		},
		{
			name:    "empty list in OR_VENUES_JSON",
			envVars: map[string]string{"OR_VENUES_JSON": `[]`, "TARGET_PLATFORM": "slack"},
		},
		{
			name:    "missing TARGET_PLATFORM",
			envVars: map[string]string{"OR_VENUES_JSON": `[{"name":"ICLR"}]`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 環境変数をセット
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}

			_, err := Load()
			if err == nil {
				t.Error("Load() should have failed, but it did not")
			}

			// 環境変数をクリーンアップ
			for key := range tc.envVars {
				os.Unsetenv(key)
			}
		})
	}
}