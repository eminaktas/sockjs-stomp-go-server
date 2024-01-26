package main

import (
	"fmt"
	"log"
	"net/http"

	stompserver "github.com/eminaktas/sockjs-stomp-go-server"
	"github.com/go-stomp/stomp/v3/frame"
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

	newEndpoint := newEndpoint(listener, "/echo/")
	go newEndpoint.Start()
	defer newEndpoint.Stop()

	for {
		msg := newEndpoint.ReadMessage()
		fmt.Println(string(msg))
	}
}

type Endpoint interface {
	Start()
	Stop()
	ReadMessage() []byte
}

type endpoint struct {
	server  stompserver.StompServer
	message chan []byte
}

func newEndpoint(listener stompserver.RawConnectionListener, appDestinationPrefix string) Endpoint {
	config := stompserver.NewStompConfig(
		60000, // 6 seconds
		[]string{"/echo"},
	)

	return &endpoint{
		server:  stompserver.NewStompServer(listener, config),
		message: make(chan []byte),
	}
}

func (e *endpoint) Start() {
	e.server.OnApplicationRequest(e.bridgeMessage)
	e.server.OnSubscribeEvent(e.bridgeAddSubscription)
	e.server.OnUnsubscribeEvent(e.bridgeRemoveSubscription)
	e.server.Start()
}

func (e *endpoint) Stop() {
	e.server.Stop()
}

func (e *endpoint) ReadMessage() []byte {
	msg := <-e.message
	return msg
}

func (e *endpoint) bridgeMessage(destination string, message []byte, connectionId string) {
	e.message <- message
}

func (e *endpoint) bridgeAddSubscription(conId string, subId string, destination string, frame *frame.Frame) {
	fmt.Println(conId, subId, destination)

}

func (e *endpoint) bridgeRemoveSubscription(conId string, subId string, destination string) {
	fmt.Println(conId, subId, destination)
}
