package rtmp

import (
	"github.com/pixelbender/go-rtmp/amf"
	"log"
	"net"
	"time"
	"sync/atomic"
	"sync"
)

var bufferSize = 4096

// A Conn represents the STUN connection and implements the RTMP protocol over net.Conn interface.
type Conn struct {
	net.Conn
	r *reader
	w *writer

	RequestTimeout time.Duration

	seq uint64
	tx map[uint64]chan[]byte
}

// NewConn creates a Conn connection on the given net.Conn
func NewConn(inner net.Conn) *Conn {
	c := &Conn{
		Conn: inner,
		r:newReader(inner),
		w:newWriter(inner),
		tx: make(map[uint64]chan[]byte),
	}
	go c.loop()
	return c
}

func (c *Conn) Connect(app string) error {
}

func (c *Conn) Request(name string, args...interface{}) (err error) {
	ch := make(chan []byte, 1)

	enc := &amf.Encoder{}
	enc.EncodeString(name)
	enc.EncodeNumber(1)
	for _, it := range args {
		if err = enc.Encode(it); err != nil {
			return
		}
	}
	enc.Bytes()
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
	sa.unpack(b[1537:3073])
	log.Printf("send %v", ch)
	log.Printf("done reading server %v %v", sh, sa)
	if sh.Proto != ch.Proto {
		return ErrHandshake
	}
	ca := &handshakeAck{
		Time:     sh.Time,
		RecvTime: uint32(rt / time.Millisecond),
	}
	log.Printf("send %v", ca)
	_, err = c.Write(ca.pack(b))
	return
}

func (c *Conn) Setup() (err error) {
	b := make([]byte, 4)
	putUint32(b, 250000)
	c.w.WriteFull(0x2, 0, msgSetChunkSize, 0, b)
	putUint32(b, 250000)
	c.w.WriteFull(0x2, 0, msgAckSize, 0, b)

	enc := &amf.Encoder{}
	enc.EncodeString("connect")
	enc.EncodeNumber(1)
	err = enc.Encode(&connectInfo{
		App:          "mgw/10",
		Capabilities: 239,
		VideoCodecs:  252,
		AudioCodecs:  3575,
		TcURL:        "rtmp://localhost/mgw/10",
	})
	c.w.WriteFull(0x3, 0, msgAmf0Command, 0, enc.Bytes())
	if err != nil {
		log.Printf(">>> %v", err)
		return
	}
	return c.w.Flush()
}

func (c *Conn) loop() {
}

func (c *Conn) ReadChunk() (ch *chunk, err error) {
	for {
		if ch, err = c.r.ReadChunk(); err != nil {
			return
		}
		switch ch.Type {
		case msgSetChunkSize:
			if len(ch.Data) == 4 {
				c.r.size = int(getUint32(ch.Data))
			}
		case msgAckSize:
			if len(ch.Data) == 4 {
				//r.ack = int(getUint32(result.Data))
			}
		case msgSetBandwidth:
			if len(ch.Data) == 5 {
				//r.bw = int(getUint32(result.Data))
				log.Printf("chunk bw = ")
			}
		default:
			return
		}
	}
}

/*
func (c *Conn) Request() error {
}

func (c *Conn) readChunk() (ch *chunk, err error) {
	var b byte
	if b, err = c.buf.ReadByte(); err != nil {
		return
	}
	fmt, id := b[0] >> 6 & 0x03, uint32(b[0] & 0x3f)
	if id == 0 {
		if b, err = c.buf.ReadByte(); err != nil {
			return
		}
		id = uint32(b) + 64
	} else if id == 1 {

		if b, err = c.buf.ReadByte(); err != nil {
			return
		}
		id = uint32(b[1]) << 8 + uint32(b[2]) + 64
	}
	if _, err = c.buf.Discard(3); err != nil {
		return
	}

	return
}

type chunk struct {

}
*/
