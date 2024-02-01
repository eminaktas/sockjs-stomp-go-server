package stompserver

import (
	"time"

	"github.com/go-stomp/stomp/v3/frame"
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
}

type rawConnResult struct {
	conn RawConnection
	err  error
}

func NewSockJSConnectionListenerFromExisting(session sockjs.Session, isClosed chan interface{}) (RawConnectionListener, error) {
	l := &sockJSConnectionListener{
		connectionChannel: make(chan rawConnResult),
	}

	go func() {
		l.connectionChannel <- rawConnResult{
			conn: &sockJSStompConnection{
				sockJS:   NewSockJSWrapper(session),
				isClosed: isClosed,
			},
		}
	}()

	return l, nil
}

func (l *sockJSConnectionListener) Accept() (RawConnection, error) {
	cr := <-l.connectionChannel
	return cr.conn, cr.err
}

// We can't close listener since don't have access to server itself.
func (l *sockJSConnectionListener) Close() error {
	return nil
}
