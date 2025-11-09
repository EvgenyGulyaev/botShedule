package main

import (
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/tg"
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/vk"
	"github.com/EvgenyGulyaev/botShedule/iternal/config"
)

func main() {
	// Загружаем конфигурацию
	c := config.LoadConfig()

	botTg := tg.GetBot(c.Env["TG_BOT_TOKEN"])
	go botTg.StartHandleMessage()

	botVk := vk.GetBot(c.Env["VK_BOT_TOKEN"])
	botVk.StartHandleMessage()

}
