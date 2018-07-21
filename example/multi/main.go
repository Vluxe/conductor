package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Vluxe/conductor"
	"github.com/acmacalister/helm"
	"github.com/acmacalister/skittles"
)

const (
	channelName = "hello"
)

type textMessage struct {
	Text string `json:"text"`
}

func main() {
	server1 := createServer(8080)
	server2 := createServer(8081)

	fmt.Printf(skittles.Cyan("started server 1 on port: %d\n"), server1.Port)
	startServer(server1)

	fmt.Printf(skittles.Yellow("started server 2 on port: %d\n"), server2.Port)
	startServer(server2)

	addSister(server1, server2)
	addSister(server2, server1)

	fmt.Println(skittles.BoldGreen("Starting reader..."))
	reader := bufio.NewReader(os.Stdin)

	client1 := createClient(server1.Port, true)
	client2 := createClient(server2.Port, false)

	client1Writing := true
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(skittles.BoldRed(err))
		}

		m := textMessage{Text: line}
		b, _ := json.Marshal(m)
		if client1Writing {
			client1.Write(channelName, b)
		} else {
			client2.Write(channelName, b)
		}
		client1Writing = !client1Writing
	}
}

func createServer(port int) *conductor.Server {
	deduper := conductor.NewDeDuper(time.Second*10, time.Second*30)
	auther := conductor.NewSimpleAuth()
	storer := conductor.NewSimpleStorage(100)
	manager := conductor.NewSisterManager()
	server := conductor.New(port, deduper, auther, storer, nil, manager)
	return server
}

func startServer(s *conductor.Server) {
	s.Start(false)
	router := helm.New(s.WebsocketHandler)
	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", s.Port), Handler: router}
	go httpServer.ListenAndServe()
}

func createClient(port int, blue bool) *conductor.Client {
	url := fmt.Sprintf("ws://localhost:%d", port)
	client, err := conductor.NewClient(url)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
	client.Bind(channelName)
	go func() {
		for {
			select {
			case message := <-client.Read:
				switch opcode := message.Opcode; opcode {
				case conductor.WriteOpcode:
					var m textMessage
					json.Unmarshal(message.Body, &m)
					if blue {
						fmt.Print(skittles.Cyan(m.Text))
					} else {
						fmt.Print(skittles.Yellow(m.Text))
					}

				default:
					break
				}

			}
		}
	}()
	return client
}

func addSister(server *conductor.Server, sister *conductor.Server) {
	url := fmt.Sprintf("ws://localhost:%d", sister.Port)
	headers := make(map[string]string)
	headers["is_sister"] = "true"
	conn := conductor.NewSisterServer(url, headers)
	err := server.AddSister(conn)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
}
