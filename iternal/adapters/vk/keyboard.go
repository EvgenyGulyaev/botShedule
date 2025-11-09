package vk

import (
	"encoding/json"

	"github.com/SevereCloud/vksdk/v3/object"
)

func getKeyboard(buttons *[]string) interface{} {
	keyboard := object.MessagesKeyboard{
		OneTime: false, // клавиатура не одноразовая
		Buttons: [][]object.MessagesKeyboardButton{},
	}

	var row []object.MessagesKeyboardButton
	for i, text := range *buttons {
		btn := object.MessagesKeyboardButton{
			Action: object.MessagesKeyboardButtonAction{
				Type:  "text",
				Label: text,
			},
		}

		row = append(row, btn)
		if (i+1)%3 == 0 {
			keyboard.Buttons = append(keyboard.Buttons, row)
			row = []object.MessagesKeyboardButton{}
		}
	}

	// Добавляем неполную последнюю строку
	if len(row) > 0 {
		keyboard.Buttons = append(keyboard.Buttons, row)
	}

	jsonBytes, _ := json.Marshal(keyboard)

	return string(jsonBytes)
}
