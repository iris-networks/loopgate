package telegram

import (
	"fmt"
	"log"
	"loopgate/internal/session"
	"loopgate/internal/types"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api            *tgbotapi.BotAPI
	sessionManager *session.Manager
	updates        tgbotapi.UpdatesChannel
}

func NewBot(token string, sessionManager *session.Manager) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	bot.Debug = false

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	return &Bot{
		api:            bot,
		sessionManager: sessionManager,
		updates:        updates,
	}, nil
}

func (b *Bot) Start() {
	log.Println("Starting Telegram bot...")
	
	for update := range b.updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (b *Bot) SendHITLRequest(request *types.HITLRequest) error {
	telegramID, err := b.sessionManager.GetTelegramID(request.ClientID)
	if err != nil {
		return fmt.Errorf("failed to get telegram ID for client %s: %w", request.ClientID, err)
	}

	var msg tgbotapi.MessageConfig

	if len(request.Options) > 0 {
		msg = b.createMessageWithButtons(telegramID, request)
	} else {
		msg = b.createSimpleMessage(telegramID, request)
	}

	sentMsg, err := b.api.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	request.TelegramMsgID = sentMsg.MessageID
	return nil
}

func (b *Bot) createMessageWithButtons(chatID int64, request *types.HITLRequest) tgbotapi.MessageConfig {
	text := fmt.Sprintf("🤖 *HITL Request*\n\n%s\n\n*Request ID:* `%s`\n*Client:* %s\n*Session:* %s",
		request.Message, request.ID, request.ClientID, request.SessionID)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, option := range request.Options {
		callback := fmt.Sprintf("response:%s:%d", request.ID, i)
		button := tgbotapi.NewInlineKeyboardButtonData(option, callback)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{button})
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg.ReplyMarkup = keyboard

	return msg
}

func (b *Bot) createSimpleMessage(chatID int64, request *types.HITLRequest) tgbotapi.MessageConfig {
	text := fmt.Sprintf("🤖 *HITL Request*\n\n%s\n\n*Request ID:* `%s`\n*Client:* %s\n*Session:* %s\n\nPlease reply with your response.",
		request.Message, request.ID, request.ClientID, request.SessionID)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	return msg
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	if message.IsCommand() {
		b.handleCommand(message)
		return
	}

	if message.ReplyToMessage != nil {
		b.handleReply(message)
	}
}

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.sendResponse(message.Chat.ID, "Welcome to Loopgate! Use /status to check active sessions.")
	case "status":
		b.handleStatusCommand(message.Chat.ID)
	case "pending":
		b.handlePendingCommand(message.Chat.ID)
	default:
		b.sendResponse(message.Chat.ID, "Unknown command. Available commands: /start, /status, /pending")
	}
}

func (b *Bot) handleStatusCommand(chatID int64) {
	sessions, err := b.sessionManager.GetActiveSessions()
	if err != nil {
		log.Printf("Error getting active sessions: %v", err)
		b.sendResponse(chatID, "Error retrieving active sessions.")
		return
	}
	
	if len(sessions) == 0 {
		b.sendResponse(chatID, "No active sessions found.")
		return
	}

	text := "*Active Sessions:*\n\n"
	for _, session := range sessions {
		if session.TelegramID == chatID {
			text += fmt.Sprintf("• Session: `%s`\n  Client: %s\n  Started: %s\n\n",
				session.ID, session.ClientID, session.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}

	b.sendMarkdownResponse(chatID, text)
}

func (b *Bot) handlePendingCommand(chatID int64) {
	pending, err := b.sessionManager.GetPendingRequests()
	if err != nil {
		log.Printf("Error getting pending requests: %v", err)
		b.sendResponse(chatID, "Error retrieving pending requests.")
		return
	}
	
	if len(pending) == 0 {
		b.sendResponse(chatID, "No pending requests.")
		return
	}

	text := "*Pending Requests:*\n\n"
	for _, request := range pending {
		telegramID, err := b.sessionManager.GetTelegramID(request.ClientID)
		if err != nil || telegramID != chatID {
			continue
		}
		
		text += fmt.Sprintf("• Request: `%s`\n  Message: %s\n  Client: %s\n\n",
			request.ID, request.Message, request.ClientID)
	}

	b.sendMarkdownResponse(chatID, text)
}

func (b *Bot) handleReply(message *tgbotapi.Message) {
	replyText := message.ReplyToMessage.Text
	
	if !strings.Contains(replyText, "Request ID:") {
		return
	}

	requestID := b.extractRequestID(replyText)
	if requestID == "" {
		return
	}

	err := b.sessionManager.UpdateRequestResponse(requestID, message.Text, true)
	if err != nil {
		b.sendResponse(message.Chat.ID, fmt.Sprintf("Error updating request: %v", err))
		return
	}

	b.sendResponse(message.Chat.ID, "✅ Response recorded successfully!")
}

func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	data := query.Data
	log.Printf("Received callback query from user %d: %s", query.From.ID, data)
	
	if !strings.HasPrefix(data, "response:") {
		log.Printf("Ignoring non-response callback: %s", data)
		return
	}

	parts := strings.Split(data, ":")
	if len(parts) != 3 {
		return
	}

	requestID := parts[1]
	optionIndex, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}

	request, err := b.sessionManager.GetRequest(requestID)
	if err != nil {
		b.answerCallbackQuery(query.ID, "Request not found")
		return
	}

	if optionIndex >= len(request.Options) {
		b.answerCallbackQuery(query.ID, "Invalid option")
		return
	}

	selectedOption := request.Options[optionIndex]
	approved := strings.ToLower(selectedOption) != "cancel" && 
	           strings.ToLower(selectedOption) != "reject" &&
	           strings.ToLower(selectedOption) != "deny"

	log.Printf("Processing response for request %s: option='%s', approved=%t", requestID, selectedOption, approved)

	err = b.sessionManager.UpdateRequestResponse(requestID, selectedOption, approved)
	if err != nil {
		log.Printf("Error updating request %s: %v", requestID, err)
		b.answerCallbackQuery(query.ID, "Error updating request")
		return
	}

	log.Printf("Successfully updated request %s with response: %s", requestID, selectedOption)

	b.answerCallbackQuery(query.ID, fmt.Sprintf("Selected: %s", selectedOption))
	
	updateText := fmt.Sprintf("✅ *Response Recorded*\n\nSelected: %s\nRequest ID: `%s`", 
		selectedOption, requestID)
	
	edit := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, updateText)
	edit.ParseMode = "Markdown"
	b.api.Send(edit)
}

func (b *Bot) extractRequestID(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Request ID:") {
			parts := strings.Split(line, "`")
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

func (b *Bot) sendResponse(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.api.Send(msg)
}

func (b *Bot) sendMarkdownResponse(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) answerCallbackQuery(queryID, text string) {
	callback := tgbotapi.NewCallback(queryID, text)
	b.api.Request(callback)
}