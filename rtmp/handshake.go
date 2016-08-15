package rtmp

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

var ErrHandshake = errors.New("rtmp: handshake error")

var salt = []byte{0xf0, 0xee, 0xc2, 0x4a, 0x80, 0x68, 0xbe, 0xe8, 0x2e, 0x00, 0xd0, 0xd1, 0x02, 0x9e, 0x7e, 0x57, 0x6e, 0xec, 0x5d, 0x2d, 0x29, 0x80, 0x6f, 0xab, 0x93, 0xb8, 0xe6, 0x36, 0xcf, 0xeb, 0x31, 0xae}
var gafp = []byte("Genuine Adobe Flash Player 001")
var gafms = []byte("Genuine Adobe Flash Media Server 001")
var gafps = append(gafp, salt...)
var gafmss = append(gafms, salt...)

type handshakeHello struct {
	Proto   uint8
	Time    uint32
	Version uint32
	Digest  []byte // 32 bytes
	PubKey  []byte
}

func (h *handshakeHello) pack(b []byte, key []byte) []byte {
	b[0] = h.Proto
	putUint32(b[1:], h.Time)
	putUint32(b[5:], h.Version)
	rand.Read(b[9:1537])
	return b[:1537]
}

func (h *handshakeHello) unpack(b []byte, key []byte) {
	h.Proto = b[0]
	h.Time = getUint32(b[1:])
	h.Version = getUint32(b[5:])
	//if !h.unpackDigest(b, key, h.offset(b, 772) % 728 + 776, 768) {
	//	h.unpackDigest(b, key, h.offset(b, 8) % 728 + 12, 1532)
	//}
}

func (h *handshakeHello) unpackDigest(b []byte, key []byte, off int, pos int) bool {
	if 0 <= off && off+32 <= len(b) {
		dig := b[off : off+32]
		crc := h.digest(b, key, off)
		if bytes.Compare(dig, crc) == 0 {
			h.Digest = crc
			off = h.offset(b, pos)
			if 0 < off && off+128 <= len(b) {
				h.PubKey = make([]byte, 128)
				copy(h.PubKey, b[off:off+128])
			}
			return true
		}
	}
	return false
}

func (h *handshakeHello) offset(b []byte, off int) int {
	return int(b[off]) + int(b[off+1]) + int(b[off+2]) + int(b[off+3])
}

func (h *handshakeHello) digest(b []byte, key []byte, off int) []byte {
	r := hmac.New(sha256.New, key)
	if off > 0 {
		r.Write(b[:off])
	}
	if off+32 < len(b) {
		r.Write(b[off+32:])
	}
	return r.Sum(nil)
}

type handshakeAck struct {
	Time     uint32
	RecvTime uint32
	Digest   []byte
}

func (a *handshakeAck) pack(b []byte) []byte {
	putUint32(b, a.Time)
	putUint32(b[4:], a.RecvTime)
	rand.Read(b[8:1536])
	return b[:1536]
}

func (a *handshakeAck) unpack(b []byte) {
	a.Time = getUint32(b)
	a.RecvTime = getUint32(b[4:])
}

func serverHandshake(c *Conn) (err error) {
	/*b := make([]byte, 3073)
	b[0] = 0x03
	ct, cv := uint32(wallclock()), uint32(0x10000000)
	putUint32(b[1:], ct)
	putUint32(b[5:], cv)
	rand.Read(b[9:1537])
	if _, err = c.Write(b); err != nil {
		return
	}

	if _, err = c.Read(b[:1537]); err != nil {
		return
	}
	if b[0] != 0x03 {
		return ErrHandshake
	}
	ct, _ := getUint32(b[1:]), getUint32(b[5:])
	st, sv := uint32(wallclock()), uint32(0x10000000)
	putUint32(b[1:], st)
	putUint32(b[5:], sv)
	putUint32(b[1537:], st)
	putUint32(b[1541:], ct)
	rand.Read(b[1545:])
	if _, err = c.Write(b); err != nil {
		return
	}
	if _, err = c.Read(b[:1537]); err != nil {
		return
	}
	return*/
	return nil
}
