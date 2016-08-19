package rtmp

type Stream struct {
	conn *Conn
	id   uint32
}

func (s *Stream) Send(name string, args ...interface{}) {
	s.conn.req.write(s.conn, s.id, name, args...)
}

func (s *Stream) Flush() error {
	return s.conn.w.Flush()
}

func (s *Stream) Play(name string) error {
	s.Send("receiveAudio", true)
	s.Send("receiveVideo", true)
	s.Send("play", name)
	return s.Flush()
}

func (s *Stream) Close() error {
	s.Send("closeStream")
	return s.Flush()
}
