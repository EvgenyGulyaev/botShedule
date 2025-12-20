package console

import (
	"github.com/EvgenyGulyaev/botShedule/pkg/logger"
	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
)

type Bot struct {
	log       *logger.Logger
	name      string
	blockUser map[int64]bool
}

func GetBot(name string) *Bot {
	return singleton.GetInstance("bot-console", func() interface{} {
		bot := &Bot{log: logger.GetLogger(), name: name}
		return bot
	}).(*Bot)
}

func (b *Bot) Publish(chatID int64, message string) {
	if b.isBlock(chatID) {
		return
	}
	b.log.Printf("[%s]%d - %s", b.name, chatID, message)
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
