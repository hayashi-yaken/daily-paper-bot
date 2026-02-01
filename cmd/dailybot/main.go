package main

import (
	"fmt"
	"log"

	"github.com/hayashi-yaken/daily-paper-bot/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
	}

	// 設定をロード
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	fmt.Println("Configuration loaded successfully.")
	fmt.Printf("Target Platform: %s\n", cfg.TargetPlatform)
	fmt.Printf("Dry Run: %v\n", cfg.DryRun)

	// TODO: ここからBotのメイン処理を呼び出す (DPB-007)
}
