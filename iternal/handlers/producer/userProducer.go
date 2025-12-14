package producer

import (
	"log"

	"github.com/EvgenyGulyaev/botShedule/pkg/broker"
)

type User struct {
	Id   int64  `json:"id"`
	Name string `json:"username"`
	Net  string `json:"network"`
}

func (u *User) Publish() bool {
	b := broker.Get()

	err := broker.Publish[User](b.Nc, "user", *u)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}
