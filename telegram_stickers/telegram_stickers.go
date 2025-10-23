package telegramstickers

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/ws117z5/telegram_bot/config"
)

type UserSession struct {
	Stickers  []telego.InputSticker
	PackName  string
	PackTitle string
}

type Bot struct {
	api      *telego.Bot
	sessions map[int64]*UserSession
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewBot(token string) (*Bot, error) {
	api, err := telego.NewBot(token, telego.WithDefaultDebugLogger())
	if err != nil {
		return nil, err
	}

	return &Bot{
		api:      api,
		sessions: make(map[int64]*UserSession),
	}, nil
}

func (b *Bot) getSession(userID int64) *UserSession {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.sessions[userID] == nil {
		b.sessions[userID] = &UserSession{
			Stickers: make([]telego.InputSticker, 0),
		}
	}
	return b.sessions[userID]
}

func (b *Bot) clearSession(userID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.sessions, userID)
}

func (b *Bot) handleStart(ctx context.Context, message telego.Message) error {
	_, _ = b.api.SendMessage(
		ctx,
		tu.Message(
			tu.ID(message.Chat.ID),
			"🎨 <b>Welcome to Sticker Pack Creator Bot!</b>\n\n"+
				"<b>Commands:</b>\n"+
				"/start - Show this message\n"+
				"/add - Start adding stickers\n"+
				"/list - View your current stickers\n"+
				"/create - Create sticker pack\n"+
				"/clear - Clear all stickers\n\n"+
				"<b>Usage:</b>\n"+
				"1. Send /add to start\n"+
				"2. Send stickers one by one\n"+
				"3. Send /create to make your pack",
		).WithParseMode("HTML"))
	return nil
}

func (b *Bot) handleAdd(ctx context.Context, message telego.Message) {
	session := b.getSession(message.From.ID)

	_, _ = b.api.SendMessage(
		ctx,
		tu.Message(
			tu.ID(message.Chat.ID),
			"✅ Ready to receive stickers!\n\n"+
				"Send me stickers one by one. When done, use /create to make your pack.\n"+
				fmt.Sprintf("Current stickers: %d", len(session.Stickers)),
		),
	)
}

func (b *Bot) handleSticker(ctx context.Context, message telego.Message) {
	session := b.getSession(message.From.ID)

	// Get the sticker file
	sticker := message.Sticker

	// Determine format based on sticker properties
	format := "static"
	if sticker.IsAnimated {
		format = "animated"
	} else if sticker.IsVideo {
		format = "video"
	}

	// Create input sticker for the new pack
	inputSticker := telego.InputSticker{
		Sticker:   telego.InputFile{FileID: sticker.FileID},
		EmojiList: []string{"😀"}, // Default emoji, can be customized
		Format:    format,
	}

	session.Stickers = append(session.Stickers, inputSticker)

	_, _ = b.api.SendMessage(
		ctx,
		tu.Message(
			tu.ID(message.Chat.ID),
			fmt.Sprintf("Sticker added! Total: %d\n\n"+
				"Send more stickers or use /create to make your pack.", len(session.Stickers)),
		),
	)
}

func (b *Bot) handleList(ctx context.Context, message telego.Message) {
	session := b.getSession(message.From.ID)

	if len(session.Stickers) == 0 {
		_, _ = b.api.SendMessage(
			ctx,
			tu.Message(
				tu.ID(message.Chat.ID),
				"You haven't added any stickers yet.\n\nUse /add and send stickers to begin.",
			))
		return
	}

	_, _ = b.api.SendMessage(
		ctx,
		tu.Message(
			tu.ID(message.Chat.ID),
			fmt.Sprintf("📋 You have %d sticker(s) ready.\n\nUse /create to make your pack!", len(session.Stickers)),
		))
}

func (b *Bot) handleCreate(ctx context.Context, message telego.Message) {
	session := b.getSession(message.From.ID)

	if len(session.Stickers) == 0 {
		_, _ = b.api.SendMessage(
			ctx,
			tu.Message(
				tu.ID(message.Chat.ID),
				"You need to add at least one sticker first.\n\nUse /add and send stickers.",
			))
		return
	}

	// Get bot username for pack name
	botUser, err := b.api.GetMe(ctx)
	if err != nil {
		log.Printf("Error getting bot info: %v", err)
		_, _ = b.api.SendMessage(
			ctx,
			tu.Message(
				tu.ID(message.Chat.ID),
				"Failed to get bot information. Please try again.",
			))
		return
	}

	// Generate unique pack name
	packName := fmt.Sprintf("pack_%d_%d_by_%s", message.From.ID, message.Date, botUser.Username)
	packTitle := fmt.Sprintf("%s's Custom Pack", message.From.FirstName)

	_, _ = b.api.SendMessage(
		ctx,
		tu.Message(
			tu.ID(message.Chat.ID),
			"Creating your sticker pack...",
		),
	)

	// Create new sticker set
	params := &telego.CreateNewStickerSetParams{
		UserID:      message.From.ID,
		Name:        packName,
		Title:       packTitle,
		Stickers:    session.Stickers,
		StickerType: "regular",
	}

	err = b.api.CreateNewStickerSet(ctx, params)
	if err != nil {
		log.Printf("Error creating sticker set: %v", err)
		_, _ = b.api.SendMessage(
			ctx,
			tu.Message(
				tu.ID(message.Chat.ID),
				fmt.Sprintf("Failed to create sticker pack: %v\n\nPlease try again.", err),
			))
		return
	}

	// Clear session after successful creation
	b.clearSession(message.From.ID)

	_, _ = b.api.SendMessage(
		ctx,
		tu.Message(
			tu.ID(message.Chat.ID),
			fmt.Sprintf("<b>Sticker pack created successfully!</b>\n\n"+
				"Pack name: <code>%s</code>\n"+
				"Title: %s\n\n"+
				"You can find it here: https://t.me/addstickers/%s",
				packName, packTitle, packName),
		).WithParseMode("HTML"))
}

func (b *Bot) handleClear(ctx context.Context, message telego.Message) {
	b.clearSession(message.From.ID)

	_, _ = b.api.SendMessage(
		ctx,
		tu.Message(
			tu.ID(message.Chat.ID),
			"All stickers cleared!\n\nUse /add to start fresh.",
		))
}

func (b *Bot) Run() error {
	// Get bot user info
	ctx := context.Background()

	user, err := b.api.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("get me: %w", err)
	}
	log.Printf("Bot started: @%s", user.Username)

	// Get updates
	updates, _ := b.api.UpdatesViaLongPolling(ctx, nil)
	// Create handler
	bh, err := th.NewBotHandler(b.api, updates)
	if err != nil {
		return fmt.Errorf("failed to create bot handler: %w", err)
	}

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		// Send a message with inline keyboard
		b.handleStart(ctx, message)
		return nil
	}, th.CommandEqual("start"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		// Send a message with inline keyboard
		b.handleAdd(ctx, message)
		return nil
	}, th.CommandEqual("add"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		// Send a message with inline keyboard
		b.handleAdd(ctx, message)
		return nil
	}, th.CommandEqual("list"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		// Send a message with inline keyboard
		b.handleCreate(ctx, message)
		return nil
	}, th.CommandEqual("create"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		// Send a message with inline keyboard
		b.handleClear(ctx, message)
		return nil
	}, th.CommandEqual("clear"))

	// Handle stickers
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		b.handleSticker(ctx, message)
		return nil
	}, th.AnyMessageWithMedia())

	bh.Start()
	defer func() { _ = bh.Stop() }()

	return nil

}

func RunStickerBot() {
	cfg := config.GetConfig()
	bot, err := NewBot(cfg.TelegramBotToken)
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
