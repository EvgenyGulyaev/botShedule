package main

import (
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/tg"
	"github.com/EvgenyGulyaev/botShedule/iternal/config"
)

func main() {
	// Загружаем конфигурацию
	c := config.LoadConfig()
	bot := tg.GetBot(c.Env["TG_BOT_TOKEN"])
	bot.StartHandleMessage()

}
