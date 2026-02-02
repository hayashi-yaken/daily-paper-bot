package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

var venuesConfigPath = "assets/venues.json"

// VenueConfig は一つの学会に関する設定を保持します。
type VenueConfig struct {
	Name  string `json:"name"`  // 表示名 (例: "ICLR")
	Venue string `json:"venue"` // API用Venue ID
	Year  int    `json:"year"`  // 年
}

// Config はアプリケーション全体の設定を保持します。
type Config struct {
	// OpenReview
	Venues []VenueConfig // 複数学会を保持

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

	// Misc
	CustomUserAgent string
}

// Load は環境変数と設定ファイルから設定を読み込み、検証します。
func Load() (*Config, error) {
	cfg := &Config{}
	var err error

	// --- ファイルからの設定 ---

	// config/venues.json から学会リストを読み込む
	bytes, err := os.ReadFile(venuesConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read venues config file at %s: %w", venuesConfigPath, err)
	}
	if err := json.Unmarshal(bytes, &cfg.Venues); err != nil {
		return nil, fmt.Errorf("failed to parse venues config file: %w", err)
	}
	if len(cfg.Venues) == 0 {
		return nil, fmt.Errorf("no venues found in %s", venuesConfigPath)
	}

	// --- 環境変数からの設定 ---

	// TargetPlatform
	cfg.TargetPlatform = os.Getenv("TARGET_PLATFORM")
	if cfg.TargetPlatform == "" {
		return nil, fmt.Errorf("environment variable TARGET_PLATFORM is required")
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

	// --- 任意項目（デフォルト値あり） ---

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

	cfg.CustomUserAgent = os.Getenv("CUSTOM_USER_AGENT")
	if cfg.CustomUserAgent == "" {
		cfg.CustomUserAgent = "daily-paper-bot/1.0 (+https://github.com/hayashi-yaken/daily-paper-bot)"
	}

	return cfg, nil
}
