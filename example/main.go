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

type serverReq struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type serverResp struct {
	Type  string `json:"type"`
	Count string `json:"count"`
}

type serverHandler struct {
	storer *conductor.SimpleStorage
}

func main() {
	deduper := conductor.NewDeDuper(time.Second*10, time.Second*30)
	auther := conductor.NewSimpleAuth()
	storer := conductor.NewSimpleStorage(100)
	handler := &serverHandler{storer: storer}
	server := conductor.New(8080, deduper, auther, storer, handler)
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
				switch opcode := message.Opcode; opcode {
				case conductor.WriteOpcode:
					var u user
					json.Unmarshal(message.Body, &u)
					fmt.Print(skittles.Cyan(u.Text))
				case conductor.ServerOpcode:
					var resp serverResp
					json.Unmarshal(message.Body, &resp)
					fmt.Print(skittles.Cyan("message history count: " + resp.Count))
				default:
					break
				}

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
	i := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(skittles.BoldRed(err))
		}

		u := user{Text: line}
		b, _ := json.Marshal(u)
		client.Write("hello", b)
		i++
		if i%2 == 0 {
			u := serverReq{Type: "history", Name: "hello"}
			b, _ := json.Marshal(u)
			client.ServerMessage(b)
		}
	}
}

func (s *serverHandler) Process(conn conductor.Connection, message *conductor.Message) {
	//fmt.Println("got a server message!")
	// var req serverReq
	// json.Unmarshal(message.Body, &req)
	// //fmt.Println("type: " + req.Type)
	// if req.Type == "history" {
	// 	messages := s.storer.Get(req.Name)
	// 	count := strconv.Itoa(len(messages))
	// 	//fmt.Println("count: " + count)
	// 	resp := serverResp{Type: "history", Count: count}
	// 	b, _ := json.Marshal(resp)
	// 	conn.Write(&conductor.Message{Opcode: conductor.ServerOpcode, ChannelName: "", Body: b})
	// }
}
