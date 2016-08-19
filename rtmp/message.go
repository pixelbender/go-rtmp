package rtmp

const (
	msgSetChunkSize = uint8(0x01)
	msgAbort        = uint8(0x02)
	msgAck          = uint8(0x03)
	msgUserControl  = uint8(0x04)
	msgAckSize      = uint8(0x05)
	msgSetBandwidth = uint8(0x06)
	msgEdge         = uint8(0x07)
	msgAudio        = uint8(0x08)
	msgVideo        = uint8(0x09)
	msgAmf3Meta     = uint8(0x0f)
	msgAmf3Shared   = uint8(0x10)
	msgAmf3Command  = uint8(0x11)
	msgAmf0Meta     = uint8(0x12)
	msgAmf0Shared   = uint8(0x13)
	msgAmf0Command  = uint8(0x14)
	msgAggregate    = uint8(0x15)
	msgMax          = uint8(0x16)
)

const (
	ctrlStreamBegin      = uint8(0x00)
	ctrlStreamEOF        = uint8(0x01)
	ctrlStreamDry        = uint8(0x02)
	ctrlStreamSetBuffer  = uint8(0x03)
	ctrlStreamIsRecorded = uint8(0x04)
	ctrlPingRequest      = uint8(0x06)
	ctrlPingResponse     = uint8(0x07)
)

const (
	limitHard    = uint8(0x00)
	limitSoft    = uint8(0x01)
	limitDynamic = uint8(0x02)
)

type ClientInfo struct {
	App          string `amf:"app"`
	FlashVer     string `amf:"flashVer"`
	Capabilities uint16 `amf:"capabilities"`
	AudioCodecs  uint16 `amf:"audioCodecs"`
	VideoCodecs  uint16 `amf:"videoCodecs"`
	//VideoFunction  uint8  `amf:"videoFunction"`
	ObjectEncoding uint8 `amf:"objectEncoding"`
	//UsingProxy     bool   `amf:"fpad"`
	SwfURL  string `amf:"swfUrl,omitempty"`
	PageUrl string `amf:"pageUrl,omitempty"`
	TcURL   string `amf:"tcUrl,omitempty"`
}
