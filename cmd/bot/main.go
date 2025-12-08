package main

import (
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/tg"
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/vk"
	"github.com/EvgenyGulyaev/botShedule/iternal/config"
	"github.com/EvgenyGulyaev/botShedule/iternal/handlers"
)

func main() {
	// Загружаем конфигурацию
	c := config.LoadConfig()

	// Запускаем брокер для сообщений из вне
	handlers.RunBroker()

	botTg := tg.GetBot(c.Env["TG_BOT_TOKEN"])
	go botTg.StartHandleMessage()

	botVk := vk.GetBot(c.Env["VK_BOT_TOKEN"])
	botVk.StartHandleMessage()

}
