package producer

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

func Publish[T any](nc *nats.Conn, subject string, d T) error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return nc.Publish(subject, data)
}
