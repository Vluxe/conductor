package main

import (
	"flag"
	"fmt"
	"github.com/Vluxe/conductor"
)

type Storage int

var addr = flag.Int("addr", 8080, "http service address")
var peer = flag.Int("peer", 8081, "http service address")

func main() {
	flag.Parse()
	var s Storage
	server := conductor.CreateServer(*addr)
	server.Notification = s
	server.AddPeer(fmt.Sprintf("ws://localhost:%d", *peer))
	server.Start()
}

func (s Storage) PersistentHandler() {
	fmt.Println("store some stuff...")
}
