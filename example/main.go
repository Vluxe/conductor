package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Vluxe/conductor"
	"github.com/acmacalister/skittles"
)

type user struct {
	Text string `json:"text"`
}

func main() {
	deduper := conductor.NewDeDuper(time.Second*10, time.Second*30)
	server := conductor.New(8080, deduper)
	go server.Start()

	go func() {
		client, err := conductor.NewClient("ws://localhost:8080")
		if err != nil {
			log.Fatal(skittles.BoldRed(err))
		}

		client.Bind("hello")
		for {
			select {
			case message := <-client.Read:
				var u user
				json.Unmarshal(message.([]byte), &u)
				fmt.Print(skittles.Cyan(u.Text))
			}
		}
	}()

	client, err := conductor.NewClient("ws://localhost:8080")
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
	client.Bind("hello")

	fmt.Println(skittles.BoldGreen("Starting reader..."))
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(skittles.BoldRed(err))
		}

		u := user{Text: line}
		b, _ := json.Marshal(u)
		client.Write("hello", b)
	}
}
