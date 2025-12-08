package handlers

import (
	"context"
	"os"

	"github.com/EvgenyGulyaev/botShedule/iternal/consumer"
	"github.com/EvgenyGulyaev/botShedule/pkg/broker"
	"github.com/EvgenyGulyaev/botShedule/pkg/shutdown"
)

func RunBroker() {
	b, err := broker.NewNatsBroker(os.Getenv("NATS_URL"))
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	listener := consumer.NewMessageListener(ctx, b.Nc)

	// Запускаем shutdown для освобождения ресурсов, при перезапуске
	sd := shutdown.Get()
	sd.Add(cancel)
	sd.Add(listener.Stop)
	sd.Add(b.Close)
	go sd.Wait()
}
