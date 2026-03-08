package main

import "github.com/goforj/events"

type UserCreated struct {
	ID string `json:"id"`
}

func main() {
	bus, err := events.NewNull()
	if err != nil {
		panic(err)
	}
	if err := bus.Publish(UserCreated{ID: "123"}); err != nil {
		panic(err)
	}
}
