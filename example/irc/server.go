package main

import (
	"fmt"
	"github.com/Vluxe/conductor"
)

type Storage int

func main() {
	var s Storage
	server := conductor.CreateServer(8080)
	server.Notification = s
	server.Start()
}

func (s Storage) PersistentHandler(message conductor.Message, token string) {
	fmt.Println("store some stuff: ", message.Body)
}

func (s Storage) BindHandler(message conductor.Message, token string) {
	fmt.Printf("%s bound to: %s\n", message.Name, message.ChannelName)
}

func (s Storage) UnBindHandler(message conductor.Message, token string) {
	fmt.Printf("%s unbound from: %s\n", message.Name, message.ChannelName)
}

func (s Storage) InviteHandler(message conductor.Message, token string) {
	fmt.Println("got an invitation.")
}
