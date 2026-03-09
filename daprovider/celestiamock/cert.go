package celestiamock

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// DACertificateMessageHeaderFlag matches Nitro's custom DA certificate header byte.
	DACertificateMessageHeaderFlag byte = 0x01

	certMagic   = "MCDA"
	certVersion = uint8(1)
	certSize    = 4 + 1 + 8 + 4 + 32 // magic + version + id + len + hash
)

var (
	ErrInvalidCertificate = errors.New("invalid mock celestia certificate")
)

// Certificate is a small self-describing pointer to payload stored in the in-memory backend.
// Binary format:
// [magic(4)][version(1)][id(8)][payloadLen(4)][sha256(32)]
type Certificate struct {
	ID         uint64
	PayloadLen uint32
	Hash       [32]byte
}

func NewCertificate(id uint64, payload []byte) Certificate {
	return Certificate{
		ID:         id,
		PayloadLen: uint32(len(payload)),
		Hash:       sha256.Sum256(payload),
	}
}

func (c Certificate) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, certSize))
	if _, err := buf.WriteString(certMagic); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(certVersion); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, c.ID); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, c.PayloadLen); err != nil {
		return nil, err
	}
	if _, err := buf.Write(c.Hash[:]); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Certificate) UnmarshalBinary(data []byte) error {
	if len(data) != certSize {
		return fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidCertificate, certSize, len(data))
	}
	if string(data[:4]) != certMagic {
		return fmt.Errorf("%w: bad magic", ErrInvalidCertificate)
	}
	if data[4] != certVersion {
		return fmt.Errorf("%w: unsupported version %d", ErrInvalidCertificate, data[4])
	}
	c.ID = binary.BigEndian.Uint64(data[5:13])
	c.PayloadLen = binary.BigEndian.Uint32(data[13:17])
	copy(c.Hash[:], data[17:49])
	return nil
}

// ExtractCertificateBytesFromSequencerMessage supports both raw cert and
// sequencer message prefixed with header byte 0x01.
func ExtractCertificateBytesFromSequencerMessage(sequencerMsg []byte) ([]byte, error) {
	if len(sequencerMsg) == 0 {
		return nil, fmt.Errorf("%w: empty message", ErrInvalidCertificate)
	}
	if sequencerMsg[0] == DACertificateMessageHeaderFlag {
		if len(sequencerMsg) < 2 {
			return nil, fmt.Errorf("%w: missing bytes after DA header", ErrInvalidCertificate)
		}
		return sequencerMsg[1:], nil
	}
	return sequencerMsg, nil
}
