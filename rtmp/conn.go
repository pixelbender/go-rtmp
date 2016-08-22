package rtmp

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

var ErrTimeout = errors.New("rtmp: i/o timeout")

var bufferSize = 4096

// A Conn represents the RTMP connection and implements the RTMP protocol over net.Conn interface.
type Conn struct {
	net.Conn
	r *reader
	w *writer

	RequestTimeout time.Duration
	req            requestMux

	str   map[int64]*Stream
	strmu sync.RWMutex
}

// NewConn creates a Conn connection on the given net.Conn
func NewConn(inner net.Conn) *Conn {
	c := &Conn{
		Conn:           inner,
		r:              newReader(inner),
		w:              newWriter(inner),
		RequestTimeout: 5 * time.Second,
		str:            make(map[int64]*Stream),
	}
	return c
}

func (c *Conn) Serve() error {
	//	var str *Stream
	for {
		ch, err := c.r.ReadChunk()
		if err != nil {
			return err
		}
		switch ch.Type {
		case msgSetChunkSize:
			if len(ch.Data) == 4 {
				c.r.size = int(be.Uint32(ch.Data))
				log.Printf("chunk size=%v", c.r.size)
			}
		case msgAckSize:
			if len(ch.Data) == 4 {
				//r.ack = int(getUint32(result.Data))
			}
		case msgSetBandwidth:
			if len(ch.Data) == 5 {
				//r.bw = int(getUint32(result.Data))
			}
		case msgAmf0Command, msgAmf3Command:
			c.req.handleChunk(ch)
		default:
			log.Printf("chunk %+v", ch)
		}
	}
}

func (c *Conn) Request(name string, args ...interface{}) (*Response, error) {
	return c.req.request(c, 0, name, args...)
}

func (c *Conn) CreateStream() (str *Stream, err error) {
	var res *Response
	if res, err = c.Request("createStream", nil); err != nil {
		return
	}
	if err = res.Skip(); err != nil {
		return
	}
	var id int64
	if id, err = res.DecodeInt(); err != nil {
		return
	}
	str = &Stream{conn: c, id: uint32(id)}
	c.strmu.Lock()
	c.str[id] = str
	c.strmu.Unlock()
	return
}

func (c *Conn) Handshake() (err error) {
	b := make([]byte, 3073)
	ch := &handshakeHello{
		Proto:   0x03,
		Time:    uint32(0),
		Version: uint32(0),
	}
	ts := time.Now()
	if _, err = c.Write(ch.pack(b, nil)); err != nil {
		return
	}
	if _, err = c.Read(b[:3073]); err != nil {
		return
	}
	rt := time.Now().Sub(ts)
	sh, sa := &handshakeHello{}, &handshakeAck{}
	sh.unpack(b[:1537], nil)

	b = b[1537:3073]
	sa.unpack(b)
	if sh.Proto != ch.Proto {
		return ErrHandshake
	}
	ca := &handshakeAck{
		Time:     sh.Time,
		RecvTime: uint32(rt / time.Millisecond),
	}
	_, err = c.Write(ca.pack(b))

	log.Printf("handshake done")

	c.w.size = 10000
	be.PutUint32(b, uint32(c.w.size))
	c.w.WriteFull(0x2, 0, msgSetChunkSize, 0, b[:4])
	be.PutUint32(b, 250000)
	c.w.WriteFull(0x2, 0, msgAckSize, 0, b[:4])

	return nil
}
