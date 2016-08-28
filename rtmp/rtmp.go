package rtmp

import (
	"net"
	"net/url"
	"strings"
)

// TODO: dial with dialer

func Dial(uri string) (*Conn, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}
	var c net.Conn
	switch strings.ToLower(u.Scheme) {
	case "rtmps":
	//...
	case "rtmp":
		if port == "" {
			port = "1935"
		}
		if c, err = net.Dial("tcp", net.JoinHostPort(host, port)); err != nil {
			return nil, err
		}
	}
	conn := NewConn(c)
	err = conn.Handshake()
	if err != nil {
		return nil, err
	}
	return conn, nil
}
