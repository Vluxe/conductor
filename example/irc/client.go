package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/Vluxe/conductor"
	"github.com/acmacalister/skittles"
	"log"
	"os"
)

var addr = flag.Int("addr", 8080, "http service address")

func main() {
	flag.Parse()
	client, err := conductor.CreateClient(fmt.Sprintf("ws://localhost:%d", *addr), "client", "")
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
	client.Bind("hello", func(message conductor.Message) {
		fmt.Printf(skittles.Cyan("%s"), message.Body)
	})
	go client.ReadLoop() //start listening for messages
	fmt.Println(skittles.BoldGreen("Connected!"))
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(skittles.BoldRed(err))
		}
		client.SendMessage(line, "hello", nil)
	}
}
