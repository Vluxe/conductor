package main

import (
	"fmt"
	"github.com/Vluxe/conductor"
)

type Storage int

func main() {
	var s Storage
	server := conductor.CreateServer()
	server.Notification = s
	server.Start()
}

func (s Storage) PersistentHandler() {
	fmt.Println("store some stuff...")
}
