package rtmp

import (
	"bytes"
	"github.com/pixelbender/go-rtmp/amf"
	"log"
	"testing"
)

func TestRTMP(t *testing.T) {
	c, err := Dial("rtmp://localhost/one/two")
	if err != nil {
		t.Fatal("dial rtmp:", err)
	}
	log.Printf("handshake done")
	c.Setup()
	for {
		ch, err := c.ReadChunk()
		if err != nil {
			t.Fatal("read chunk:", err)
		}
		if ch.Type == msgAmf0Command {
			dec := amf.NewDecoder(bytes.NewReader(ch.Data))
			for {
				v, err := dec.DecodeNext()
				if err != nil {
					t.Fatal("decode error:", err)
				}
				log.Printf("decode: %#v", v)
			}
		}
		log.Printf("chunk %#v", ch)
	}
}
