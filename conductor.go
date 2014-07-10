// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// code borrowed from example chat program of
// https://github.com/gorilla/websocket

package conductor

import (
	"flag"
	"github.com/acmacalister/skittles"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")

// Start up the core websocket server...
func CreateServer() {
	log.Println(skittles.Cyan("starting server..."))
	go h.run()
	http.HandleFunc("/", serveWs)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
}
