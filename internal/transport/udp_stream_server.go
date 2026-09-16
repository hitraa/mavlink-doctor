package transport

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/bluenviron/gomavlib/v4"
)

// udpVirtualConn wraps per-client UDP communication as a stream net.Conn.
type udpVirtualConn struct {
	serverConn *net.UDPConn
	remoteAddr *net.UDPAddr
	pipeR      *io.PipeReader
	pipeW      *io.PipeWriter
	closed     chan struct{}
	closeOnce  sync.Once
}

func newUDPVirtualConn(serverConn *net.UDPConn, remoteAddr *net.UDPAddr) *udpVirtualConn {
	pr, pw := io.Pipe()
	return &udpVirtualConn{
		serverConn: serverConn,
		remoteAddr: remoteAddr,
		pipeR:      pr,
		pipeW:      pw,
		closed:     make(chan struct{}),
	}
}

func (c *udpVirtualConn) Read(b []byte) (n int, err error) {
	return c.pipeR.Read(b)
}

func (c *udpVirtualConn) Write(b []byte) (n int, err error) {
	return c.serverConn.WriteToUDP(b, c.remoteAddr)
}

func (c *udpVirtualConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.pipeW.Close()
		_ = c.pipeR.Close()
	})
	return nil
}

func (c *udpVirtualConn) LocalAddr() net.Addr {
	return c.serverConn.LocalAddr()
}

func (c *udpVirtualConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *udpVirtualConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *udpVirtualConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *udpVirtualConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// udpStreamListener implements net.Listener over a UDP server socket,
// demuxing packets from each remote sender into a virtual stream connection.
type udpStreamListener struct {
	conn       *net.UDPConn
	acceptChan chan net.Conn
	clientsMu  sync.Mutex
	clients    map[string]*udpVirtualConn
	closed     chan struct{}
	closeOnce  sync.Once
}

func newUDPStreamListener(listenAddr string) (*udpStreamListener, error) {
	laddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}

	l := &udpStreamListener{
		conn:       conn,
		acceptChan: make(chan net.Conn, 16),
		clients:    make(map[string]*udpVirtualConn),
		closed:     make(chan struct{}),
	}

	go l.readLoop()
	return l, nil
}

func (l *udpStreamListener) readLoop() {
	buf := make([]byte, 65535)

	for {
		n, remoteAddr, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-l.closed:
				return
			default:
				l.Close()
				return
			}
		}

		key := remoteAddr.String()
		l.clientsMu.Lock()
		client, exists := l.clients[key]
		if !exists {
			client = newUDPVirtualConn(l.conn, remoteAddr)
			l.clients[key] = client
			select {
			case l.acceptChan <- client:
			default:
			}
		}
		l.clientsMu.Unlock()

		// Write payload bytes to the client pipe
		_, _ = client.pipeW.Write(buf[:n])
	}
}

func (l *udpStreamListener) Accept() (net.Conn, error) {
	select {
	case conn, ok := <-l.acceptChan:
		if !ok {
			return nil, errors.New("listener closed")
		}
		return conn, nil
	case <-l.closed:
		return nil, errors.New("listener closed")
	}
}

func (l *udpStreamListener) Close() error {
	l.closeOnce.Do(func() {
		close(l.closed)
		_ = l.conn.Close()
		l.clientsMu.Lock()
		for _, c := range l.clients {
			_ = c.Close()
		}
		l.clientsMu.Unlock()
	})
	return nil
}

func (l *udpStreamListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

// NewUDPStreamServer creates a gomavlib endpoint using custom UDP stream listener
// with IsDatagram=false so chunked packets sent to the server are reassembled as a byte stream.
func NewUDPStreamServer(listenAddress string, port int) gomavlib.Endpoint {
	addr := net.JoinHostPort(listenAddress, fmt.Sprint(port))
	return &gomavlib.EndpointCustomServer{
		Listen: func() (net.Listener, error) {
			return newUDPStreamListener(addr)
		},
		Label:      "udp-stream-server:" + addr,
		IsDatagram: false,
	}
}
