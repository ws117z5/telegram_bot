package main

import (
	"log"

	"github.com/ws117z5/telegram_bot/config"
	telegramstickers "github.com/ws117z5/telegram_bot/telegram_stickers"
)

func main() {
	cfg := config.GetConfig()

	bot, err := telegramstickers.NewBot(cfg.TelegramBotToken)

	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	log.Println("Starting bot...")
	if err := bot.Run(); err != nil {
		log.Fatalf("Bot error: %v", err)
	}

	// Keep running
	select {}
}
