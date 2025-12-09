package vk

import (
	"github.com/EvgenyGulyaev/botShedule/iternal/formatter"
	"github.com/SevereCloud/vksdk/v3/api/params"
)

func (b *Bot) createMessage(m *formatter.HandlerMessage) *params.MessagesSendBuilder {
	res, ks := m.GetAnswer()
	mes := b.makeMessage(m.ChatId, res)
	if len(ks) != 0 {
		mes.Keyboard(getKeyboard(&ks))
	}

	return mes
}

func (b *Bot) makeMessage(chatID int64, message string) *params.MessagesSendBuilder {
	mes := params.NewMessagesSendBuilder()
	mes.Message(message)
	mes.RandomID(0)
	mes.PeerID(int(chatID))
	return mes
}
