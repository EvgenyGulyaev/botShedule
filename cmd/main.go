package main

import (
	"github.com/EvgenyGulyaev/botShedule/iternal/adapters/bot"
	"github.com/EvgenyGulyaev/botShedule/iternal/config"
)

func main() {
	// Загружаем конфигурацию
	c := config.LoadConfig()
	bot.GetBot(c.Env["BOT_TOKEN"])
}
