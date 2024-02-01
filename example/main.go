package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	stompserver "github.com/eminaktas/sockjs-stomp-go-server"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/gorilla/websocket"
	"github.com/igm/sockjs-go/v3/sockjs"
)

const ConnectionEndpoint = "/connect"

var allowedOrigins = []string{}

func main() {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}

	opts := sockjs.DefaultOptions
	opts.CheckOrigin = checkOrigin
	opts.WebsocketUpgrader = &upgrader

	http.Handle("/connect/", sockjs.NewHandler(ConnectionEndpoint, opts, connectionHandler))
	log.Println("Server started on port: 8085")
	log.Fatal(http.ListenAndServe(":8085", nil))
}

func connectionHandler(s sockjs.Session) {
	// We need to be notified when the client drops the connection.
	// This reason we use channel for it.
	isClosed := make(chan interface{})

	listener, err := stompserver.NewSockJSConnectionListenerFromExisting(s, isClosed)
	if err != nil {
		fmt.Println(err)
		return
	}

	newEndpoint := newEndpoint(listener, []string{"/echo/"})
	go newEndpoint.Start()

	// Writes message to STOMP subscribe destination.
	go func(newEndpoint Endpoint) {
		for {
			select {
			case <-isClosed:
				fmt.Println("Write message stopped due to client connection gone")
				return
			default:
				msg := []byte("repeated message")
				newEndpoint.WriteMessage("/topic", msg)
				fmt.Println("Outgoing message:", string(msg))
				time.Sleep(5 * time.Second)
			}
		}
	}(newEndpoint)

	// Prints the message send via connection.
	for {
		select {
		case <-isClosed:
			fmt.Println("Message send stopped due to client connection gone")
			return
		case msg := <-newEndpoint.ReadMessage():
			fmt.Println("Incoming message:", string(msg))
		default:
			continue
		}
	}
}

func checkOrigin(r *http.Request) bool {
	if len(allowedOrigins) == 0 {
		return true
	}

	origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}
	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}

	for _, allowedOrigin := range allowedOrigins {
		if strings.EqualFold(u.Host, allowedOrigin) {
			return true
		}
	}

	return false
}

type Endpoint interface {
	Start()
	Stop()
	ReadMessage() chan []byte
	WriteMessage(string, []byte)
}

type endpoint struct {
	server  stompserver.StompServer
	message chan []byte
}

func newEndpoint(listener stompserver.RawConnectionListener, appDestinationPrefix []string) Endpoint {
	config := stompserver.NewStompConfig(
		60000, // 6 seconds
		appDestinationPrefix,
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

	fmt.Println("Connection started by client")
	e.server.Start()
}

func (e *endpoint) Stop() {
	fmt.Println("Connection stopped by client")
	e.server.Stop()
}

func (e *endpoint) ReadMessage() chan []byte {
	return e.message
}

func (e *endpoint) WriteMessage(destination string, message []byte) {
	e.server.SendMessage(destination, message)
}

func (e *endpoint) bridgeMessage(destination string, message []byte, connectionId string) {
	e.message <- message
}

func (e *endpoint) bridgeAddSubscription(conId string, subId string, destination string, frame *frame.Frame) {
	fmt.Println("bridgeAddSubscription:", conId, subId, destination)

}

func (e *endpoint) bridgeRemoveSubscription(conId string, subId string, destination string) {
	fmt.Println("bridgeRemoveSubscription:", conId, subId, destination)
}
