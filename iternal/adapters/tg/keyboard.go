package tg

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetKeyboard(ks *[]string) interface{} {
	if len(*ks) == 0 {
		return tgbotapi.NewRemoveKeyboard(true)
	}
	var rows [][]tgbotapi.KeyboardButton
	var currentRow []tgbotapi.KeyboardButton

	for i, item := range *ks {
		currentRow = append(currentRow, tgbotapi.NewKeyboardButton(item))

		if (i+1)%3 == 0 {
			rows = append(rows, currentRow)
			currentRow = []tgbotapi.KeyboardButton{}
		}
	}

	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	return tgbotapi.NewReplyKeyboard(rows...)
}
