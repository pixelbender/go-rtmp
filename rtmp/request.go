package rtmp

import (
	"fmt"
	"github.com/pixelbender/go-rtmp/amf"
	"log"
	"sync"
	"time"
)

type Response struct {
	amf.Decoder
	name string
}

func (req *Response) error() error {
	if req.name == "_error" {
		return &ErrorResponse{}
	}
	return nil
}

type ErrorResponse struct {
	amf.Decoder
}

func (err *ErrorResponse) Error() string {
	params := make([]interface{}, 0, 10)
	for {
		var v interface{}
		if err.Decode(&v) != nil {
			break
		}
		params = append(params, v)
	}
	return fmt.Sprintf("rtmp: error response %+v", params)
}

type requestMux struct {
	seq int64
	req map[int64]chan *Response
	mu  sync.Mutex
}

func (r *requestMux) handleChunk(ch *chunk) error {
	d := make([]byte, len(ch.Data))
	copy(d, ch.Data)

	// TODO: peeker

	dec := amf.NewDecoderBytes(0, d)
	name, err := dec.DecodeString()
	if err != nil {
		return err
	}
	id, err := dec.DecodeInt()
	if err != nil {
		return err
	}

	tx, res := r.getRequest(id), &Response{dec, name}
	if tx != nil {
		select {
		case tx <- res:
		default:
		}
	} else {
		log.Printf("unhandled: %v", id)
		for {
			var p interface{}
			err := res.Decode(&p)
			if err != nil {
				break
			}
			log.Printf("%+v", p)
		}
	}

	return nil
}

func (r *requestMux) newRequest() (id int64, tx chan *Response) {
	tx = make(chan *Response, 1)
	r.mu.Lock()
	r.seq++
	id = r.seq
	if r.req == nil {
		r.req = make(map[int64]chan *Response)
	}
	r.req[id] = tx
	r.mu.Unlock()
	return
}

func (r *requestMux) getRequest(id int64) (tx chan *Response) {
	r.mu.Lock()
	tx = r.req[id]
	delete(r.req, id)
	r.mu.Unlock()
	return
}

func (r *requestMux) deleteRequest(id int64) {
	r.mu.Lock()
	delete(r.req, id)
	r.mu.Unlock()
}

func (r *requestMux) write(c *Conn, str uint32, name string, args ...interface{}) error {
	enc := amf.NewEncoder(0)
	enc.EncodeString(name)
	enc.EncodeInt(0)
	enc.EncodeNull()
	for _, it := range args {
		if err := enc.Encode(it); err != nil {
			return err
		}
	}
	c.w.WriteFull(0x3, 0, msgAmf0Command, str, enc.Bytes())
	return nil
}

func (r *requestMux) request(c *Conn, str uint32, name string, args ...interface{}) (res *Response, err error) {
	id, tx := r.newRequest()
	defer r.deleteRequest(id)

	log.Printf("%s(%v) %+v", name, str, args)

	enc := amf.NewEncoder(0)
	enc.EncodeString(name)
	enc.EncodeInt(id)
	for _, it := range args {
		if err = enc.Encode(it); err != nil {
			return
		}
	}
	c.w.WriteFull(0x3, 0, msgAmf0Command, str, enc.Bytes())

	if err = c.w.Flush(); err != nil {
		return
	}

	select {
	case res, _ = <-tx:
		if res == nil {
			err = ErrTimeout
		} else {
			err = res.error()
		}
	case <-time.After(c.RequestTimeout):
		err = ErrTimeout
	}

	return
}
