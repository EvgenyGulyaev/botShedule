package consumer

import (
	"context"

	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/console"
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/tg"
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/vk"
	"github.com/EvgenyGulyaev/botShedule/iternal/config"
	"github.com/EvgenyGulyaev/botShedule/pkg/broker"
	"github.com/EvgenyGulyaev/botShedule/pkg/logger"
	"github.com/nats-io/nats.go"
)

type Message struct {
	User    int64  `json:"user"`
	Message string `json:"message"`
	Network string `json:"network"`
}

type MessageSender interface {
	Publish(chatID int64, message string)
}

type MessageListener struct {
	ctx       context.Context
	nc        *nats.Conn
	cancelSub context.CancelFunc
	log       *logger.Logger
}

func NewMessageListener(ctx context.Context, nc *nats.Conn) *MessageListener {
	ol := &MessageListener{
		ctx: ctx,
		nc:  nc,
		log: logger.GetLogger(),
	}
	ol.start()
	return ol
}

func (ol *MessageListener) start() {
	ch, cancelSub, err := broker.Subscribe[Message](ol.nc, "message", 100, ol.ctx)
	if err != nil {
		ol.log.Printf("MessageListener subscribe error: %v", err)
		return
	}

	ol.cancelSub = cancelSub

	// Запускаем обработку в фоне
	go func() {
		defer cancelSub()
		for order := range ch {
			ol.handleMessage(order)
		}
	}()
}

func (ol *MessageListener) handleMessage(m Message) {
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
	bot.Publish(m.User, m.Message)
}

func (ol *MessageListener) Stop() {
	if ol.cancelSub != nil {
		ol.cancelSub()
	}
}
