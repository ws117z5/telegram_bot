package main

import (
	"log"

	telegramstickers "github.com/ws117z5/telegram_bot/telegram_stickers"
)

func main() {
	bot, err := telegramstickers.NewBot("7113196065:AAEenTOKBuC1FnTrw5K8koozuaNlKe2UKdY")

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
