package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	// t.Setenv is used to set environment variables for the duration of a test.
	// It automatically restores the original value when the test and any of its subtests complete.

	t.Run("success with slack", func(t *testing.T) {
		t.Setenv("OR_VENUE", "ICLR.cc/2025/Conference")
		t.Setenv("OR_YEAR", "2025")
		t.Setenv("TARGET_PLATFORM", "slack")
		t.Setenv("SLACK_BOT_TOKEN", "test-token")
		t.Setenv("SLACK_CHANNEL_ID", "test-channel")
		t.Setenv("DRY_RUN", "true")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected no error, but got: %v", err)
		}
		if cfg.Venue != "ICLR.cc/2025/Conference" {
			t.Errorf("unexpected venue: got %v, want %v", cfg.Venue, "ICLR.cc/2025/Conference")
		}
		if cfg.Year != 2025 {
			t.Errorf("unexpected year: got %v, want %v", cfg.Year, 2025)
		}
		if !cfg.DryRun {
			t.Errorf("unexpected dry_run: got %v, want %v", cfg.DryRun, true)
		}
		// Check default values
		if cfg.SelectStrategy != "random" {
			t.Errorf("unexpected select_strategy: got %v, want %v", cfg.SelectStrategy, "random")
		}
		if cfg.AbstractMaxChars != 1200 {
			t.Errorf("unexpected abstract_max_chars: got %v, want %v", cfg.AbstractMaxChars, 1200)
		}
		// Check default user agent
		defaultUserAgent := "daily-paper-bot/1.0 (+https://github.com/hayashi-yaken/daily-paper-bot)"
		if cfg.CustomUserAgent != defaultUserAgent {
			t.Errorf("unexpected user_agent: got %v, want %v", cfg.CustomUserAgent, defaultUserAgent)
		}
	})

	t.Run("success with discord and custom defaults", func(t *testing.T) {
		t.Setenv("OR_VENUE", "CVPR.cc/2024/Conference")
		t.Setenv("OR_YEAR", "2024")
		t.Setenv("TARGET_PLATFORM", "discord")
		t.Setenv("DISCORD_WEBHOOK_URL", "http://discord.example.com")
		t.Setenv("SELECT_STRATEGY", "latest")
		t.Setenv("ABSTRACT_MAX_CHARS", "500")
		t.Setenv("CUSTOM_USER_AGENT", "my-custom-agent/2.0")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected no error, but got: %v", err)
		}
		if cfg.TargetPlatform != "discord" {
			t.Errorf("unexpected target_platform: got %v, want %v", cfg.TargetPlatform, "discord")
		}
		if cfg.SelectStrategy != "latest" {
			t.Errorf("unexpected select_strategy: got %v, want %v", cfg.SelectStrategy, "latest")
		}
		if cfg.AbstractMaxChars != 500 {
			t.Errorf("unexpected abstract_max_chars: got %v, want %v", cfg.AbstractMaxChars, 500)
		}
		if cfg.CustomUserAgent != "my-custom-agent/2.0" {
			t.Errorf("unexpected user_agent: got %v, want %v", cfg.CustomUserAgent, "my-custom-agent/2.0")
		}
	})

	t.Run("error on missing required field", func(t *testing.T) {
		// OR_VENUE is missing
		t.Setenv("OR_YEAR", "2025")
		t.Setenv("TARGET_PLATFORM", "slack")
		t.Setenv("SLACK_BOT_TOKEN", "test-token")
		t.Setenv("SLACK_CHANNEL_ID", "test-channel")

		_, err := Load()
		if err == nil {
			t.Fatal("expected an error, but got none")
		}
	})

	t.Run("error on missing slack config", func(t *testing.T) {
		t.Setenv("OR_VENUE", "ICLR.cc/2025/Conference")
		t.Setenv("OR_YEAR", "2025")
		t.Setenv("TARGET_PLATFORM", "slack")
		// SLACK_BOT_TOKEN is missing

		_, err := Load()
		if err == nil {
			t.Fatal("expected an error, but got none")
		}
	})

	t.Run("error on invalid year", func(t *testing.T) {
		t.Setenv("OR_VENUE", "ICLR.cc/2025/Conference")
		t.Setenv("OR_YEAR", "not-a-year")
		t.Setenv("TARGET_PLATFORM", "slack")
		t.Setenv("SLACK_BOT_TOKEN", "test-token")
		t.Setenv("SLACK_CHANNEL_ID", "test-channel")

		_, err := Load()
		if err == nil {
			t.Fatal("expected an error, but got none")
		}
	})
}
