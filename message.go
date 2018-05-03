package conductor

import (
	"bytes"
	"encoding/binary"
)

const (
	BindOpcode        = iota // BindOpcode bind to a channel. This will create the channel if it does not exist.
	UnbindOpcode             // UnbindOpcode unbind from a channel.
	WriteOpcode              // WriteOpcode broadcasts on provided channel.
	ServerOpcode             // ServerOpcode intend to be between a single client and the server (not broadcasted).
	CleanUpOpcode            // a message to cleanup a disconnected client/connection.
	StreamStartOpcode        // a message to start streaming chucks of data (like a file)
	StreamEndOpcode          // a mess
	StreamWriteOpcode        //TBD
)

// Message represents the framing of the messages that get sent back and forth.
type Message struct {
	Opcode      uint16 `json:"opcode"`
	uuidSize    uint16 `json:"uuid_size"`
	Uuid        string `json:"uuid"`
	nameSize    uint16 `json:"name_size"`
	ChannelName string `json:"stream_name"`
	bodySize    uint64 `json:"body_size"`
	Body        []byte `json:"body"`
}

//Marshal converts the Message struct into bytes to transmit over a connection.
func (m *Message) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, m.Opcode); err != nil {
		return []byte(""), err
	}

	m.uuidSize = uint16(len(m.Uuid))
	if err := binary.Write(buf, binary.LittleEndian, m.uuidSize); err != nil {
		return []byte(""), err
	}
	if _, err := buf.WriteString(m.Uuid); err != nil {
		return []byte(""), err
	}

	m.nameSize = uint16(len(m.ChannelName))
	if err := binary.Write(buf, binary.LittleEndian, m.nameSize); err != nil {
		return []byte(""), err
	}
	if _, err := buf.WriteString(m.ChannelName); err != nil {
		return []byte(""), err
	}

	m.bodySize = uint64(len(m.Body))
	if err := binary.Write(buf, binary.LittleEndian, m.bodySize); err != nil {
		return []byte(""), err
	}
	if _, err := buf.Write(m.Body); err != nil {
		return []byte(""), err
	}

	return buf.Bytes(), nil
}

//Unmarshal converts a slice of bytes into a Message struct.
func Unmarshal(b []byte) (*Message, error) {
	var m Message
	buf := bytes.NewBuffer(b)
	if err := binary.Read(buf, binary.LittleEndian, &m.Opcode); err != nil {
		return nil, err
	}

	var err error
	if m.Uuid, m.uuidSize, err = readString(buf); err != nil {
		return nil, err
	}

	if m.ChannelName, m.nameSize, err = readString(buf); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.LittleEndian, &m.bodySize); err != nil {
		return nil, err
	}

	str := make([]byte, m.bodySize)
	buf.Read(str)
	m.Body = str

	return &m, nil
}

func readString(buf *bytes.Buffer) (string, uint16, error) {
	var size uint16
	if err := binary.Read(buf, binary.LittleEndian, &size); err != nil {
		return "", 0, err
	}
	str := make([]byte, size)
	buf.Read(str)
	return string(str), size, nil
}
