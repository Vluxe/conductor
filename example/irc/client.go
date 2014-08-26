package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/Vluxe/conductor"
	"github.com/acmacalister/skittles"
	"log"
	"math/rand"
	"os"
	"time"
)

var addr = flag.Int("addr", 8080, "http service address")

type color func(interface{}) string

func main() {
	flag.Parse()
	client, err := conductor.CreateClient(fmt.Sprintf("ws://localhost:%d", *addr), "blah", "")
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
	go reader(&client)
	writer(&client)
}

func writer(client *conductor.Client) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(skittles.BoldGreen("Connected!"))
	fmt.Printf("Enter your name: ")
	n, err := reader.ReadString('\n')
	name := string(n)[:len(n)-1]
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}

	message := conductor.Message{Name: name, Body: "joined the chat\n", ChannelName: "hello", OpCode: conductor.BindOpCode}
	err = client.Writer(&message)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(skittles.BoldRed(err))
		}

		opcode := conductor.WriteOpCode

		message := conductor.Message{Name: name, Body: string(line), ChannelName: "hello", OpCode: opcode}
		err = client.Writer(&message)
		if err != nil {
			log.Fatal(skittles.BoldRed(err))
		}
	}
}

func reader(client *conductor.Client) {
	userColorMap := make(map[string]color)
	for {
		message, err := client.Reader()
		if err != nil {
			log.Fatal(err)
		}
		el, ok := userColorMap[message.Name]
		if !ok {
			userColorMap[message.Name] = randomColor()
			el = userColorMap[message.Name]
		}
		if message.OpCode == conductor.UnBindOpCode {
			fmt.Printf(el("%s: left the chat\n"), message.Name)
		} else {
			fmt.Printf(el("%s: %s\n"), message.Name, message.Body)
		}
	}
}

func randomColor() color {
	rand.Seed(time.Now().UTC().UnixNano())
	random := rand.Intn(6-1) + 1
	switch random {
	case 1:
		return skittles.Cyan
	case 2:
		return skittles.Blue
	case 3:
		return skittles.Red
	case 4:
		return skittles.Green
	case 5:
		return skittles.Magenta
	case 6:
		return skittles.Yellow
	default:
		return skittles.Cyan
	}
}
