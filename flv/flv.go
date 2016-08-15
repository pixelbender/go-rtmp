package flv

type Header struct {
	Signature uint32
	Version   uint8
	Flags     uint8
}

func NewHeader(flags uint8) *Header {
	return &Header{sign, 1, flags}
}

type Tag struct {
	Type   uint8
	Size   int
	Time   int64
	Stream uint32
}

const (
	TypeAudio = uint8(0x8)
	TypeVideo = uint8(0x9)
	TypeData  = uint8(0x12)
)

const sign = uint32(0x464C56)
