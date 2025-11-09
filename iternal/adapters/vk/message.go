package vk

import (
	"github.com/EvgenyGulyaev/botShedule/iternal/formatter"
	"github.com/SevereCloud/vksdk/v3/api/params"
)

func (b *Bot) createMessage(m *formatter.HandlerMessage) *params.MessagesSendBuilder {
	res, ks := m.GetAnswer()
	mes := params.NewMessagesSendBuilder()
	mes.Message(res)
	mes.RandomID(0)
	mes.PeerID(int(m.ChatId))
	if len(ks) != 0 {
		mes.Keyboard(getKeyboard(&ks))
	}

	return mes
}
