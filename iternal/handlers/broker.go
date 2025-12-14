package handlers

import (
	"context"
	"log"
	"os"

	"github.com/EvgenyGulyaev/botShedule/pkg/broker"
	"github.com/EvgenyGulyaev/botShedule/pkg/shutdown"
	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
)

type Broker struct {
	broker.NatsBroker // расширяем внешний пакет
}

// Get Обертка - синглтон, чтобы жить на одном соединении
func Get() *Broker {
	return singleton.GetInstance("broker", func() interface{} {
		b, err := broker.NewNatsBroker(os.Getenv("NATS_URL"))
		if err != nil {
			log.Fatalf("Can't start broker, %s", err)
		}
		return &Broker{NatsBroker: *b}
	}).(*Broker)
}

func RunBroker() {
	b := Get()
	ctx, cancel := context.WithCancel(context.Background())
	listener := NewMessageListener(ctx, b.Nc)

	// Запускаем shutdown для освобождения ресурсов, при перезапуске
	sd := shutdown.Get()
	sd.Add(cancel)
	sd.Add(listener.Stop)
	sd.Add(b.Close)
	go sd.Wait()
}
