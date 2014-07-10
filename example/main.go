package main

import (
	"fmt"
	"github.com/Vluxe/conductor"
	"net/http"
)

type Auth int
type Storage int

func main() {
	var a Auth
	var s Storage
	conductor.ServeAuth(a)
	conductor.ServePersistent(s)
	conductor.CreateServer()
}

func (a Auth) InitalAuthHander(r *http.Request) bool {
	fmt.Println("cause I'm good!")
	return true
}

func (a Auth) ChannelAuthHander() bool {
	fmt.Println("channel auth")
	return true
}

func (a Auth) MessageAuthHander() bool {
	fmt.Println("message auth!")
	return true
}

func (s Storage) PersistentHandler() {
	fmt.Println("store some stuff...")
}
