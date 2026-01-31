package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config はアプリケーション全体の設定を保持します。
type Config struct {
	// OpenReview
	Venue string
	Year  int

	// Target Platform
	TargetPlatform string

	// Slack
	SlackBotToken  string
	SlackChannelID string

	// Discord
	DiscordWebhookURL string

	// Selector
	SelectStrategy   string
	AbstractMaxChars int
	DryRun           bool
}

// Load は環境変数から設定を読み込み、検証します。
func Load() (*Config, error) {
	cfg := &Config{}

	// 文字列型の必須項目
	cfg.Venue = os.Getenv("OR_VENUE")
	if cfg.Venue == "" {
		return nil, fmt.Errorf("environment variable OR_VENUE is required")
	}
	cfg.TargetPlatform = os.Getenv("TARGET_PLATFORM")
	if cfg.TargetPlatform == "" {
		return nil, fmt.Errorf("environment variable TARGET_PLATFORM is required")
	}

	// 整数型の必須項目
	yearStr := os.Getenv("OR_YEAR")
	if yearStr == "" {
		return nil, fmt.Errorf("environment variable OR_YEAR is required")
	}
	var err error
	cfg.Year, err = strconv.Atoi(yearStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OR_YEAR: %w", err)
	}

	// プラットフォームに応じた必須項目
	switch cfg.TargetPlatform {
	case "slack":
		cfg.SlackBotToken = os.Getenv("SLACK_BOT_TOKEN")
		cfg.SlackChannelID = os.Getenv("SLACK_CHANNEL_ID")
		if cfg.SlackBotToken == "" || cfg.SlackChannelID == "" {
			return nil, fmt.Errorf("SLACK_BOT_TOKEN and SLACK_CHANNEL_ID are required for slack platform")
		}
	case "discord":
		cfg.DiscordWebhookURL = os.Getenv("DISCORD_WEBHOOK_URL")
		if cfg.DiscordWebhookURL == "" {
			return nil, fmt.Errorf("DISCORD_WEBHOOK_URL is required for discord platform")
		}
	default:
		return nil, fmt.Errorf("invalid TARGET_PLATFORM: %s. must be 'slack' or 'discord'", cfg.TargetPlatform)
	}

	// 任意項目（デフォルト値あり）
	cfg.SelectStrategy = os.Getenv("SELECT_STRATEGY")
	if cfg.SelectStrategy == "" {
		cfg.SelectStrategy = "random"
	}

	abstractMaxCharsStr := os.Getenv("ABSTRACT_MAX_CHARS")
	if abstractMaxCharsStr == "" {
		cfg.AbstractMaxChars = 1200
	} else {
		cfg.AbstractMaxChars, err = strconv.Atoi(abstractMaxCharsStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ABSTRACT_MAX_CHARS: %w", err)
		}
	}

	dryRunStr := os.Getenv("DRY_RUN")
	if dryRunStr == "" {
		cfg.DryRun = false
	} else {
		cfg.DryRun, err = strconv.ParseBool(dryRunStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DRY_RUN: %w", err)
		}
	}

	return cfg, nil
}
