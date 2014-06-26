package conductor

import (
	"github.com/acmacalister/skittles"
	"log"
	"net/http"
)

func CreateServer() {
	log.Println(skittles.Cyan("starting server..."))
	http.HandleFunc("/", serveWs)
	err := http.ListenAndServe("8080", nil)
	if err != nil {
		log.Fatal(skittles.BoldRed(err))
	}
}
