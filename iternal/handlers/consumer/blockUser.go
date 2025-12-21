package consumer

import (
	"log"

	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/console"
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/tg"
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/vk"
	"github.com/EvgenyGulyaev/botShedule/iternal/config"
)

type BlockUser struct {
	User int64  `json:"user"`
	Net  string `json:"net"`
}

type MessageBlocker interface {
	Block(chatID int64)
}

func HandleBlockUser(m BlockUser) {
	c := config.LoadConfig()
	var bot MessageBlocker
	switch m.Net {
	case "vk":
		bot = vk.GetBot(c.Env["VK_BOT_TOKEN"])
	case "tg":
		bot = tg.GetBot(c.Env["TG_BOT_TOKEN"])
	default:
		bot = console.GetBot("MessageSender")
	}
	log.Println(m.Net, bot)

	bot.Block(m.User)
}
