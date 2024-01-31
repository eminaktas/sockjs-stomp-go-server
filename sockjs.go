package stompserver

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-stomp/stomp/v3/frame"
	"github.com/gorilla/websocket"
	"github.com/igm/sockjs-go/v3/sockjs"
)

type sockJSStompConnection struct {
	sockJS   *SockJSWrapper
	isClosed chan interface{}
}

func (c *sockJSStompConnection) ReadFrame() (*frame.Frame, error) {
	frameR := frame.NewReader(c.sockJS)
	f, e := frameR.Read()
	return f, e
}

func (c *sockJSStompConnection) WriteFrame(f *frame.Frame) error {
	frameWr := frame.NewWriter(c.sockJS)
	err := frameWr.Write(f)
	if err != nil {
		return err
	}
	return err
}

func (c *sockJSStompConnection) SetReadDeadline(t time.Time) {
	c.sockJS.SetReadDeadline(t)
}

func (c *sockJSStompConnection) Close() error {
	select {
	case <-c.isClosed: // already closed
	default:
		close(c.isClosed)
	}
	return c.sockJS.Close()
}

type sockJSConnectionListener struct {
	connectionChannel chan rawConnResult
	allowedOrigins    []string
}

type rawConnResult struct {
	conn RawConnection
	err  error
}

func NewSockJSConnectionListenerFromExisting(rw http.ResponseWriter, r *http.Request,
	endpoint string, allowedOrigins []string, isClosed chan interface{}) (RawConnectionListener, error) {
	l := &sockJSConnectionListener{
		connectionChannel: make(chan rawConnResult),
		allowedOrigins:    allowedOrigins,
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     l.checkOrigin,
	}

	opts := sockjs.DefaultOptions
	opts.CheckOrigin = l.checkOrigin
	opts.WebsocketUpgrader = &upgrader

	// These are not working as expected, this reason disabled.
	opts.DisableXHRStreaming = true
	opts.DisableXHR = true
	opts.DisableEventSource = true
	opts.DisableHtmlFile = true
	opts.DisableJSONP = true

	handler := sockjs.NewHandler(endpoint, opts, func(s sockjs.Session) {
		l.connectionChannel <- rawConnResult{
			conn: &sockJSStompConnection{
				sockJS:   NewSockJSWrapper(s),
				isClosed: isClosed,
			},
		}
	})

	if strings.Contains(r.URL.Path, "/info") || r.Method == "OPTIONS" {
		handler.ServeHTTP(rw, r)
		return nil, nil
	}

	go handler.ServeHTTP(rw, r)

	return l, nil
}

func (l *sockJSConnectionListener) checkOrigin(r *http.Request) bool {
	if len(l.allowedOrigins) == 0 {
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

	for _, allowedOrigin := range l.allowedOrigins {
		if strings.EqualFold(u.Host, allowedOrigin) {
			return true
		}
	}

	return false
}

func (l *sockJSConnectionListener) Accept() (RawConnection, error) {
	cr := <-l.connectionChannel
	return cr.conn, cr.err
}

// There is no way to close listener since we are using ServeHTTP here.
func (l *sockJSConnectionListener) Close() error {
	return nil
}
