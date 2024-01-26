package main

import (
	"fmt"
	"log"
	"net/http"

	stompserver "github.com/eminaktas/sockjs-stomp-go-server"
)

func main() {
	http.HandleFunc("/connect/", newHandler)
	log.Println("Server started on port: 8085")
	log.Fatal(http.ListenAndServe(":8085", nil))
}

func newHandler(wr http.ResponseWriter, r *http.Request) {
	listener, err := stompserver.NewSockJSConnectionListenerFromExisting(
		wr, r, "/connect", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	if listener == nil {
		return
	}

	config := stompserver.NewStompConfig(
		60000, // 6 seconds
		[]string{"/echo"},
	)

	server := stompserver.NewStompServer(listener, config)
	server.Start()
	defer server.Stop()
}
