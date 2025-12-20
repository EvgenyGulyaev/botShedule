package handlers

import (
	"context"

	"github.com/EvgenyGulyaev/botShedule/iternal/handlers/consumer"
	"github.com/EvgenyGulyaev/botShedule/pkg/broker"
	"github.com/EvgenyGulyaev/botShedule/pkg/shutdown"
)

func RunBroker() {
	b := broker.Get()
	ctx, cancel := context.WithCancel(context.Background())
	msgListener := broker.NewListener[consumer.Message](ctx, b.Nc, "message", consumer.HandleMessage)
	blockListener := broker.NewListener[consumer.BlockUser](ctx, b.Nc, "user.block", consumer.HandleBlockUser)

	// Запускаем shutdown для освобождения ресурсов, при перезапуске
	sd := shutdown.Get()
	sd.Add(cancel)
	sd.Add(msgListener.Stop)
	sd.Add(blockListener.Stop)
	sd.Add(b.Close)
	go sd.Wait()
}
