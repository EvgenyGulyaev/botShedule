package consumer

import (
	"log"

	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/console"
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/tg"
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/vk"
	"github.com/EvgenyGulyaev/botShedule/iternal/config"
)

type Message struct {
	User    int64  `json:"user"`
	Message string `json:"message"`
	Network string `json:"network"`
}

type MessageSender interface {
	Publish(chatID int64, message string)
}

func HandleMessage(m Message) {
	log.Println("Start Message Handler", m)
	c := config.LoadConfig()
	var bot MessageSender

	switch m.Network {
	case "vk":
		bot = vk.GetBot(c.Env["VK_BOT_TOKEN"])
	case "tg":
		bot = tg.GetBot(c.Env["TG_BOT_TOKEN"])
	default:
		bot = console.GetBot("MessageSender")
	}
	log.Println(bot)

	bot.Publish(m.User, m.Message)
}
