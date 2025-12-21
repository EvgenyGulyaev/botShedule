package tg

import (
	"log"

	"github.com/EvgenyGulyaev/botShedule/iternal/formatter"
	"github.com/EvgenyGulyaev/botShedule/iternal/handlers/producer"
	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot Структура бота
type Bot struct {
	bot       *tgbotapi.BotAPI
	isStarted bool
	blockUser map[int64]bool
}

func GetBot(botToken string) *Bot {
	return singleton.GetInstance("bot-tg", func() interface{} {
		bot, err := initBot(botToken)
		if err != nil {
			log.Fatal("Can't start bot")
		}
		return bot
	}).(*Bot)
}

func initBot(botToken string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, err
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	b := &Bot{bot: bot, blockUser: make(map[int64]bool)}
	return b, nil
}

func (b *Bot) Publish(chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)
	_, err := b.bot.Send(msg)
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) StartHandleMessage() {
	if b.isStarted {
		log.Fatal("Error, Bot is started!")
	}
	b.isStarted = true

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			user := producer.User{
				Name:  update.Message.From.UserName,
				Id:    update.Message.Chat.ID,
				Net:   "tg",
				Text:  update.Message.Text,
				MesId: update.Message.MessageID,
			}

			go user.Publish()

			if b.isBlock(update.Message.Chat.ID) {
				continue
			}

			m := &formatter.HandlerMessage{
				UserName: update.Message.From.UserName,
				From:     update.Message.Text,
				ChatId:   update.Message.Chat.ID,
				Type:     formatter.Tg,
			}

			mes, keys := m.GetAnswer()
			msg := tgbotapi.NewMessage(m.ChatId, mes)
			msg.ReplyToMessageID = update.Message.MessageID
			msg.ReplyMarkup = getKeyboard(&keys)

			_, err := b.bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (b *Bot) Block(chatID int64) {
	_, ok := b.blockUser[chatID]
	if ok {
		delete(b.blockUser, chatID)
	}
	b.blockUser[chatID] = true
}

func (b *Bot) isBlock(chatID int64) bool {
	_, ok := b.blockUser[chatID]
	return ok
}
