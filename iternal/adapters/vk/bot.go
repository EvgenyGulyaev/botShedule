package vk

import (
	"context"
	"log"

	"github.com/EvgenyGulyaev/botShedule/iternal/formatter"
	"github.com/EvgenyGulyaev/botShedule/pkg/logger"
	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
	"github.com/SevereCloud/vksdk/v3/api"
	"github.com/SevereCloud/vksdk/v3/events"
	"github.com/SevereCloud/vksdk/v3/longpoll-bot"
)

// Структура бота
type Bot struct {
	bot *longpoll.LongPoll
	api *api.VK
}

func GetBot(botToken string) *Bot {
	return singleton.GetInstance("bot-vk", func() interface{} {
		bot, err := initBot(botToken)
		if err != nil {
			log.Fatal("Can't start bot")
		}
		return bot
	}).(*Bot)
}

func initBot(botToken string) (*Bot, error) {
	api := api.NewVK(botToken)

	gs, err := api.GroupsGetByID(nil)
	if err != nil {
		return nil, err
	}

	bot, err := longpoll.NewLongPoll(api, gs.Groups[0].ID)
	if err != nil {
		return nil, err
	}

	return &Bot{bot: bot, api: api}, nil
}

func (b *Bot) StartHandleMessage() {
	log := logger.GetLogger()
	b.bot.MessageNew(func(ctx context.Context, obj events.MessageNewObject) {
		log.Printf("[%d] %s", obj.Message.FromID, obj.Message.Text)

		m := &formatter.HandlerMessage{
			UserName: b.getUserName(obj.Message.FromID),
			From:     obj.Message.Text,
			ChatId:   int64(obj.Message.PeerID),
			Type:     formatter.Vk,
		}

		mes := b.createMessage(m)

		_, err := b.api.MessagesSend(mes.Params)
		if err != nil {
			log.Printf("Ошибка при отправке сообщения: %s", err)
		}
	})

	if err := b.bot.Run(); err != nil {
		log.Fatalf("Long Poll завершился с ошибкой: %s", err)
	}
}
