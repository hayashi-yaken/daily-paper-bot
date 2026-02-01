//go:build integration

package notifier

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

// TestMain は、このパッケージ内のテストが実行される前に一度だけ呼ばれる特別な関数です。
// インテグレーションテストの前に .env ファイルを読み込むために使用します。
func TestMain(m *testing.M) {
	// ----------------------------------------------------------------
	// .env ファイルの読み込み処理
	// ----------------------------------------------------------------
	log.Println("--- Setting up for integration tests ---")

	// 実行時のカレントディレクトリを取得
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get working directory: %v", err)
	}
	log.Printf("Current working directory: %s", wd)

	// go.mod ファイルを基準にプロジェクトルートを探す
	// テストがどこから実行されても .env を見つけられるようにする
	var projectRoot string
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			projectRoot = dir
			break
		}
		parent := filepath.Dir(dir)
		if dir == parent { // ルートディレクトリまで到達
			projectRoot = ""
			break
		}
		dir = parent
	}

	if projectRoot == "" {
		log.Println("Warning: Could not find project root (go.mod). Relying on exported env vars.")
	} else {
		envPath := filepath.Join(projectRoot, ".env")
		log.Printf("Attempting to load .env file from: %s", envPath)
		if err := godotenv.Load(envPath); err != nil {
			log.Printf("Warning: Failed to load .env file: %v. Relying on exported env vars.", err)
		} else {
			log.Println(".env file loaded successfully.")
		}
	}

	// --- デバッグ用に読み込まれた値を出力 ---
	log.Printf("[DEBUG] SLACK_BOT_TOKEN set: %t", os.Getenv("SLACK_BOT_TOKEN") != "")
	log.Printf("[DEBUG] SLACK_CHANNEL_ID: '%s'", os.Getenv("SLACK_CHANNEL_ID"))
	log.Printf("[DEBUG] DISCORD_WEBHOOK_URL set: %t", os.Getenv("DISCORD_WEBHOOK_URL") != "")
	log.Println("--------------------------------------")

	// パッケージ内のテストを実行
	exitCode := m.Run()

	// テスト終了
	os.Exit(exitCode)
}