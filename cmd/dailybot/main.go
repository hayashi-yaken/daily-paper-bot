package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/hayashi-yaken/daily-paper-bot/internal/config"
	"github.com/hayashi-yaken/daily-paper-bot/internal/formatter"
	"github.com/hayashi-yaken/daily-paper-bot/internal/notifier"
	"github.com/hayashi-yaken/daily-paper-bot/internal/openreview"
	"github.com/hayashi-yaken/daily-paper-bot/internal/selector"
	"github.com/hayashi-yaken/daily-paper-bot/internal/storage"
	"github.com/joho/godotenv"
)

func main() {
	// .envファイルを読み込む（ファイルが存在しなくてもエラーにはならない）
	_ = godotenv.Load()

	if err := run(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}
	log.Println("INFO: Process completed successfully.")
}

func run() error {
	// 1. 設定を読み込み
	log.Println("INFO: Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. 各コンポーネントを初期化
	log.Println("INFO: Initializing components...")
	orClient := openreview.NewClient(cfg.CustomUserAgent)
	jsonStorage, err := storage.NewJSONStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	paperSelector := selector.NewRandomSelector(jsonStorage.IsPosted)

	var paperNotifier notifier.Notifier
	switch cfg.TargetPlatform {
	case "slack":
		paperNotifier = notifier.NewSlackNotifier(cfg.SlackBotToken, cfg.SlackChannelID)
		log.Println("INFO: Notifier set to Slack.")
	case "discord":
		paperNotifier = notifier.NewDiscordNotifier(cfg.DiscordWebhookURL)
		log.Println("INFO: Notifier set to Discord.")
	default:
		return fmt.Errorf("invalid target platform: %s", cfg.TargetPlatform)
	}

	// 3. OpenReviewから論文一覧を取得
	log.Printf("INFO: Fetching papers from OpenReview (Venue: %s)...", cfg.Venue)
	notes, err := orClient.GetNotes(cfg.Venue)
	if err != nil {
		return fmt.Errorf("failed to get notes from openreview: %w", err)
	}
	log.Printf("INFO: Fetched %d papers.", len(notes))

	// 4. 論文を選定
	// []openreview.Note を []selector.Paper に変換
	papers := make([]selector.Paper, len(notes))
	for i := range notes {
		papers[i] = &notes[i] // ポインタを格納
	}

	log.Println("INFO: Selecting a paper...")
	selectedPaper, err := paperSelector.Select(papers)
	if err != nil {
		if errors.Is(err, selector.ErrNoCandidates) {
			log.Println("INFO: No new papers to post. Nothing to do.")
			return nil // 候補なしは正常終了
		}
		return fmt.Errorf("failed to select paper: %w", err)
	}
	log.Printf("INFO: Selected paper: %s (ID: %s)", selectedPaper.GetTitle(), selectedPaper.GetID())

	// 5. 投稿メッセージを生成
	// selector.Paper を *openreview.Note に型アサーション
	selectedNote, ok := selectedPaper.(*openreview.Note)
	if !ok {
		return fmt.Errorf("selected paper is not of type *openreview.Note")
	}
	message := formatter.FormatPaper(selectedNote, cfg.Venue, cfg.Year, cfg.AbstractMaxChars)

	// 6. DryRun または 投稿 & 記録
	if cfg.DryRun {
		log.Println("INFO: Dry run mode is enabled. Skipping post and save.")
		log.Printf("--- Message to be posted ---\n%s\n--------------------------", message)
		return nil
	}

	log.Printf("INFO: Posting to %s...", cfg.TargetPlatform)
	if err := paperNotifier.Post(message); err != nil {
		return fmt.Errorf("failed to post notification: %w", err)
	}
	log.Println("INFO: Post successful.")

	log.Println("INFO: Saving posted record...")
	jsonStorage.Add(selectedPaper.GetID(), cfg.Venue)
	if err := jsonStorage.Save(); err != nil {
		return fmt.Errorf("failed to save posted record: %w", err)
	}
	log.Println("INFO: Record saved.")

	return nil
}