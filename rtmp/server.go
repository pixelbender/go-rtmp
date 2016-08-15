package rtmp

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"
)

// Server represents a RTMP server.
type Server struct {
}

func (srv *Server) ListenAndServe(network, addr string) error {
	switch network {
	case "tcp", "tcp4", "tcp6":
		l, err := net.Listen(network, addr)
		if err != nil {
			return err
		}
		return srv.Serve(tcpKeepAliveListener{l.(*net.TCPListener)})
	}
	return fmt.Errorf("stun: listen unsupported network %v", network)
}

func (srv *Server) ListenAndServeTLS(network, addr, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	l, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	l = tls.NewListener(tcpKeepAliveListener{l.(*net.TCPListener)}, config)
	return srv.Serve(l)
}

// Multiple goroutines may invoke Serve on the same Listener simultaneously.
func (srv *Server) Serve(l net.Listener) error {
	for {
		c, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(time.Millisecond)
				continue
			}
			return err
		}
		go srv.serveConn(c)
	}
}

func (srv *Server) serveConn(conn net.Conn) error {
	c := NewConn(conn)
	defer c.Close()
	//err := clientHandshake(c)
	var err error
	if err != nil {
		log.Printf("handshake: %v", err)
	} else {
		log.Printf("handshake done")
	}
	return nil
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (l tcpKeepAliveListener) Accept() (net.Conn, error) {
	c, err := l.AcceptTCP()
	if err != nil {
		return nil, err
	}
	c.SetKeepAlive(true)
	c.SetKeepAlivePeriod(time.Minute)
	return c, nil
}
