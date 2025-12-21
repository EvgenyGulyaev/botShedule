package console

import (
	"github.com/EvgenyGulyaev/botShedule/iternal/cache"
	"github.com/EvgenyGulyaev/botShedule/pkg/logger"
	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
)

type Bot struct {
	log       *logger.Logger
	name      string
	BlockUser *cache.BlockUser
}

func GetBot(name string) *Bot {
	return singleton.GetInstance("bot-console", func() interface{} {
		bot := &Bot{log: logger.GetLogger(), name: name, BlockUser: cache.InitBlockUser()}
		return bot
	}).(*Bot)
}

func (b *Bot) Publish(chatID int64, message string) {
	if b.BlockUser.IsBlock(chatID) {
		return
	}
	b.log.Printf("[%s]%d - %s", b.name, chatID, message)
}
