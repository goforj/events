package main

import (
	"context"
	"fmt"

	"github.com/goforj/events"
)

type UserCreated struct {
	ID string `json:"id"`
}

func main() {
	bus, err := events.NewSync()
	if err != nil {
		panic(err)
	}
	_, err = bus.Subscribe(func(ctx context.Context, event UserCreated) error {
		fmt.Println("received", event.ID, ctx != nil)
		return nil
	})
	if err != nil {
		panic(err)
	}
	if err := bus.Publish(UserCreated{ID: "123"}); err != nil {
		panic(err)
	}
}
