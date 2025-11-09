package vk

import (
	"fmt"

	"github.com/SevereCloud/vksdk/v3/api"
)

func (b *Bot) getUserName(id int) string {
	userIDs := []int{id}

	// Выполняем запрос к users.get
	users, err := b.api.UsersGet(api.Params{
		"user_ids": userIDs,
		"fields":   "first_name,last_name",
	})
	if err != nil {
		return ""
	}

	if len(users) > 0 {
		return fmt.Sprintf("%s %s", users[0].FirstName, users[0].LastName)
	}

	return ""
}
