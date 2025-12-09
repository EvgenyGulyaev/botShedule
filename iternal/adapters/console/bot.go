package console

import (
	"github.com/EvgenyGulyaev/botShedule/pkg/logger"
	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
)

type Bot struct {
	log  *logger.Logger
	name string
}

func GetBot(name string) *Bot {
	return singleton.GetInstance("bot-console", func() interface{} {
		bot := &Bot{log: logger.GetLogger(), name: name}
		return bot
	}).(*Bot)
}

func (b *Bot) Publish(chatID int64, message string) {
	b.log.Printf("[%s]%d - %s", b.name, chatID, message)
}
