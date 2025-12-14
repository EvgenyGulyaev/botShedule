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
	listener := consumer.NewMessageListener(ctx, b.Nc)

	// Запускаем shutdown для освобождения ресурсов, при перезапуске
	sd := shutdown.Get()
	sd.Add(cancel)
	sd.Add(listener.Stop)
	sd.Add(b.Close)
	go sd.Wait()
}
