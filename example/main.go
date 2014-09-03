package main

import (
	"flag"
	"fmt"
	"github.com/Vluxe/conductor"
	"log"
	"net/http"
)

type Storage int
type Query int
type Auther int

var addr = flag.Int("addr", 8080, "http service address")
var peer = flag.Int("peer", 8081, "http service address")

func main() {
	flag.Parse()
	var s Storage
	var q Query
	var a Auther
	server := conductor.CreateServer(*addr)
	server.Notification = s
	server.ServerQuery = q
	server.Auth = a
	peerUrl := fmt.Sprintf("ws://localhost:%d", *peer)
	log.Println("peerUrl: ", peerUrl)
	server.AuthToken = "FakeToken"
	server.AddPeer(peerUrl)
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

func (q Query) QueryHandler(message conductor.Message, token string) conductor.Message {
	fmt.Println("got a query: ", message.Body)
	return conductor.Message{Body: "nobody\n", ChannelName: "", OpCode: conductor.ServerOpCode}
}

func (a Auther) InitalAuthHandler(r *http.Request, token string) (bool, string) {
	return true, "Bob"
}
func (a Auther) ChannelAuthHandler(message conductor.Message, token string) bool {
	return true
}
func (a Auther) MessageAuthHandler(message conductor.Message, token string) bool {
	return true
}
