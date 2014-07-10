package main

import (
	"fmt"
	"github.com/Vluxe/conductor"
)

type Auth int

func main() {
	var a Auth
	conductor.ServeAuth(a)
	conductor.CreateServer()
}

func (a Auth) InitalAuthHander() bool {
	fmt.Println("cause I'm good!")
	return true
}
