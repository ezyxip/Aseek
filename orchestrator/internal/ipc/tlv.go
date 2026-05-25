package ipc

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	Magic    = 0x4152
	Version  = 1
	HeaderSize = 16
)

type MessageType uint32

const (
	TypeQuery      MessageType = 1
	TypeToken      MessageType = 2
	TypeError      MessageType = 3
	TypeBusy       MessageType = 4
	TypeDone       MessageType = 5
	TypeCancel     MessageType = 6
	TypePing       MessageType = 7
	TypePong       MessageType = 8
	TypeProfileList   MessageType = 9
	TypeProfileSwitch MessageType = 10
	TypeSources    MessageType = 11
)

type Header struct {
	Magic     uint16
	Version   uint16
	Type      MessageType
	Length    uint32
	RequestID uint32
}

func (h *Header) Encode() []byte {
	buf := make([]byte, HeaderSize)
	binary.BigEndian.PutUint16(buf[0:2], h.Magic)
	binary.BigEndian.PutUint16(buf[2:4], h.Version)
	binary.BigEndian.PutUint32(buf[4:8], uint32(h.Type))
	binary.BigEndian.PutUint32(buf[8:12], h.Length)
	binary.BigEndian.PutUint32(buf[12:16], h.RequestID)
	return buf
}

func DecodeHeader(data []byte) (*Header, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("header too short: %d", len(data))
	}

	h := &Header{
		Magic:     binary.BigEndian.Uint16(data[0:2]),
		Version:   binary.BigEndian.Uint16(data[2:4]),
		Type:      MessageType(binary.BigEndian.Uint32(data[4:8])),
		Length:    binary.BigEndian.Uint32(data[8:12]),
		RequestID: binary.BigEndian.Uint32(data[12:16]),
	}

	if h.Magic != Magic {
		return nil, fmt.Errorf("bad magic: 0x%04x", h.Magic)
	}
	if h.Version != Version {
		return nil, fmt.Errorf("unsupported version: %d", h.Version)
	}

	return h, nil
}

type Message struct {
	Header    Header
	Payload   []byte
}

func ReadMessage(r io.Reader) (*Message, error) {
	headerBuf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, headerBuf); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	h, err := DecodeHeader(headerBuf)
	if err != nil {
		return nil, err
	}

	msg := &Message{Header: *h}

	if h.Length > 0 {
		msg.Payload = make([]byte, h.Length)
		if _, err := io.ReadFull(r, msg.Payload); err != nil {
			return nil, fmt.Errorf("read payload: %w", err)
		}
	}

	return msg, nil
}

func (m *Message) Encode() []byte {
	m.Header.Length = uint32(len(m.Payload))
	m.Header.Magic = Magic
	m.Header.Version = Version
	header := m.Header.Encode()
	return append(header, m.Payload...)
}

func NewMessage(typ MessageType, reqID uint32, payload []byte) *Message {
	return &Message{
		Header: Header{
			Magic:     Magic,
			Version:   Version,
			Type:      typ,
			Length:    uint32(len(payload)),
			RequestID: reqID,
		},
		Payload: payload,
	}
}