package vk

import (
	"context"
	"log"

	"github.com/EvgenyGulyaev/botShedule/iternal/formatter"
	"github.com/EvgenyGulyaev/botShedule/iternal/handlers/producer"
	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
	"github.com/SevereCloud/vksdk/v3/api"
	"github.com/SevereCloud/vksdk/v3/events"
	"github.com/SevereCloud/vksdk/v3/longpoll-bot"
)

// Структура бота
type Bot struct {
	bot       *longpoll.LongPoll
	api       *api.VK
	blockUser map[int64]bool
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
	b.bot.MessageNew(func(ctx context.Context, obj events.MessageNewObject) {
		userName := b.getUserName(obj.Message.FromID)
		user := producer.User{
			Name: userName,
			Id:   int64(obj.Message.PeerID),
			Net:  "vk",
			Text: obj.Message.Text,
		}
		go user.Publish()

		if b.isBlock(int64(obj.Message.PeerID)) {
			return
		}

		m := &formatter.HandlerMessage{
			UserName: userName,
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

func (b *Bot) Publish(chatID int64, message string) {
	mes := b.makeMessage(chatID, message)
	_, err := b.api.MessagesSend(mes.Params)
	if err != nil {
		log.Println(err)
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
