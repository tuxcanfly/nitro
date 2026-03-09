package celestiamock

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
)

var ErrNotFound = errors.New("mock celestia payload not found")

type record struct {
	payload []byte
	cert    Certificate
}

// Store is a threadsafe in-memory backend for certificates and payloads.
type Store struct {
	mu     sync.RWMutex
	nextID uint64
	byID   map[uint64]record
}

func NewStore() *Store {
	return &Store{
		nextID: 1,
		byID:   make(map[uint64]record),
	}
}

func (s *Store) Put(_ context.Context, payload []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID
	s.nextID++
	cp := append([]byte(nil), payload...)
	cert := NewCertificate(id, cp)
	s.byID[id] = record{payload: cp, cert: cert}
	return cert.MarshalBinary()
}

func (s *Store) GetPayloadByCert(_ context.Context, certBytes []byte) ([]byte, error) {
	var cert Certificate
	if err := cert.UnmarshalBinary(certBytes); err != nil {
		return nil, err
	}

	s.mu.RLock()
	rec, ok := s.byID[cert.ID]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	if uint32(len(rec.payload)) != cert.PayloadLen {
		return nil, fmt.Errorf("%w: payload length mismatch", ErrInvalidCertificate)
	}
	if sha256.Sum256(rec.payload) != cert.Hash {
		return nil, fmt.Errorf("%w: payload hash mismatch", ErrInvalidCertificate)
	}
	return append([]byte(nil), rec.payload...), nil
}

func (s *Store) Exists(_ context.Context, certBytes []byte) (bool, error) {
	var cert Certificate
	if err := cert.UnmarshalBinary(certBytes); err != nil {
		return false, err
	}
	s.mu.RLock()
	_, ok := s.byID[cert.ID]
	s.mu.RUnlock()
	return ok, nil
}
