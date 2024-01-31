package stompserver

import (
	"time"

	"github.com/igm/sockjs-go/v3/sockjs"
)

const (
	NullByte     = 0x00
	LineFeedByte = 0x0a
)

type SockJSReader struct {
	readerBuffer []byte
	session      sockjs.Session
	// readDeadline is not used at the moment.
	readDeadline time.Time
}

func (r *SockJSReader) Read(p []byte) (int, error) {
	// if we have no more data, read the next message from the websocket
	if len(r.readerBuffer) == 0 {
		msg, err := r.session.Recv()
		if err != nil {
			return 0, err
		}
		r.readerBuffer = []byte(msg)
	}

	n := copy(p, r.readerBuffer)
	r.readerBuffer = r.readerBuffer[n:]
	return n, nil
}

type SockJSWriter struct {
	writeBuffer []byte
	session     sockjs.Session
}

func (w *SockJSWriter) Write(p []byte) (int, error) {
	var err error
	w.writeBuffer = append(w.writeBuffer, p...)
	// if we reach a null byte or the entire message is a linefeed (heartbeat), send the message
	if p[len(p)-1] == NullByte || (len(w.writeBuffer) == 1 && len(p) == 1 && p[0] == LineFeedByte) {
		err = w.session.Send(string(w.writeBuffer))
		w.writeBuffer = []byte{}
	}
	return len(p), err
}

func NewSockJSWrapper(session sockjs.Session) *SockJSWrapper {
	return &SockJSWrapper{
		SockJSReader: &SockJSReader{
			session: session,
		},
		SockJSWriter: &SockJSWriter{
			session: session,
		},
		session: session,
	}
}

// SockJSWrapper embeds both the SockJSReader and SockJSWriter
type SockJSWrapper struct {
	*SockJSReader
	*SockJSWriter
	session sockjs.Session
}

func (w *SockJSWrapper) SetReadDeadline(t time.Time) error {
	w.readDeadline = t
	return nil
}

func (w *SockJSWrapper) Close() error {
	return w.session.Close(1000, "CLOSE_NORMAL")
}
