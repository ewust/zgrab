package ssh

import (
	"encoding/binary"
	"errors"
	"math/big"
	"strings"
)

type mpint struct {
	big.Int
}

func (mp *mpint) Marshal() ([]byte, error) {
	b := mp.Bytes()
	if len(b) == 0 {
		return nil, errors.New("Cannot encode empty mpint")
	}
	if b[0] < 0x80 {
		return b, nil
	}
	out := make([]byte, len(b)+1)
	out[0] = 0x00
	copy(out[1:], b)
	return out, nil
}

// Packet represents an SSH binary packet. See RFC
type packet struct {
	packetLength  uint32
	paddingLength uint8
	msgType       byte
	payload       []byte
	padding       []byte
	mac           []byte
}

func expectedLength(packetLength uint32, macLength uint32) int {
	totalLength := packetLength + macLength + 4
	return int(totalLength)
}

// NameList represents the name-list structure described in RFC 4251.
// See https://tools.ietf.org/html/rfc4251 for details.
type NameList []string

func (n *NameList) MarshaledLength() int {
	// 4 bytes for encoding the length
	length := 4
	nameCount := len(*n)

	// No body if its empty
	if nameCount <= 0 {
		return length
	}

	// 1 byte per comma
	length += nameCount - 1

	// Add in lengths of string
	for _, val := range *n {
		length += len(val)
	}
	return length
}

func (n *NameList) MarshalInto(dest []byte) ([]byte, error) {
	b := dest
	if len(b) < 4 {
		return dest, errShortBuffer
	}
	b = b[4:]
	joined := strings.Join(*n, ",")
	if len(b) < len(joined) {
		return dest, errShortBuffer
	}
	length := len(joined)
	binary.BigEndian.PutUint32(dest, uint32(length))
	copy(b, joined[:])
	b = b[length:]
	return b, nil
}

// Unmarshal a NameList from a byte slice of the form
// [length:body:extra] where extra is optional. Returns [extra], true
// when successful, and raw, false when unsuccessful.
func (n *NameList) Unmarshal(raw []byte) ([]byte, bool) {
	b := raw
	if len(b) < 4 {
		return raw, false
	}
	length := binary.BigEndian.Uint32(b)
	b = b[4:]
	if uint32(len(b)) < length {
		return raw, false
	}
	s := string(b[0:length])
	if len(s) == 0 {
		*n = make([]string, 0)
	} else {
		*n = NameList(strings.Split(s, ","))
	}
	b = b[length:]
	return b, true
}
