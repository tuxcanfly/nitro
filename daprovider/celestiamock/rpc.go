package celestiamock

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/server_api"
)

type Server struct {
	store          *Store
	supported      hexutil.Bytes
	maxMessageSize int
}

func NewServer(store *Store) *Server {
	if store == nil {
		store = NewStore()
	}
	return &Server{
		store:          store,
		supported:      hexutil.Bytes{DACertificateMessageHeaderFlag},
		maxMessageSize: 32 * 1024 * 1024,
	}
}

func (s *Server) SetMaxMessageSize(max int) {
	if max > 0 {
		s.maxMessageSize = max
	}
}

// daprovider_store(message, timeout)
func (s *Server) Store(message hexutil.Bytes, _ hexutil.Uint64) (*server_api.StoreResult, error) {
	if len(message) > s.maxMessageSize {
		return nil, daprovider.ErrMessageTooLarge
	}
	cert, err := s.store.Put(context.Background(), message)
	if err != nil {
		return nil, err
	}
	return &server_api.StoreResult{SerializedDACert: cert}, nil
}

// daprovider_getSupportedHeaderBytes()
func (s *Server) GetSupportedHeaderBytes(context.Context) (*server_api.SupportedHeaderBytesResult, error) {
	return &server_api.SupportedHeaderBytesResult{HeaderBytes: s.supported}, nil
}

// daprovider_getMaxMessageSize()
func (s *Server) GetMaxMessageSize(context.Context) (*server_api.MaxMessageSizeResult, error) {
	return &server_api.MaxMessageSizeResult{MaxSize: s.maxMessageSize}, nil
}

// daprovider_recoverPayload(batchNum, batchBlockHash, sequencerMsg)
func (s *Server) RecoverPayload(ctx context.Context, _ hexutil.Uint64, _ common.Hash, sequencerMsg hexutil.Bytes) (*daprovider.PayloadResult, error) {
	certBytes, err := ExtractCertificateBytesFromSequencerMessage(sequencerMsg)
	if err != nil {
		return nil, err
	}
	payload, err := s.store.GetPayloadByCert(ctx, certBytes)
	if err != nil {
		return nil, err
	}
	return &daprovider.PayloadResult{Payload: payload}, nil
}

// daprovider_collectPreimages(batchNum, batchBlockHash, sequencerMsg)
func (s *Server) CollectPreimages(context.Context, hexutil.Uint64, common.Hash, hexutil.Bytes) (*daprovider.PreimagesResult, error) {
	// Keep it simple: return empty preimages. Enough for local integration/testing.
	return &daprovider.PreimagesResult{Preimages: make(daprovider.PreimagesMap)}, nil
}

// daprovider_recoverPayloadAndPreimages(batchNum, batchBlockHash, sequencerMsg)
func (s *Server) RecoverPayloadAndPreimages(ctx context.Context, batchNum hexutil.Uint64, batchBlockHash common.Hash, sequencerMsg hexutil.Bytes) (*daprovider.PayloadAndPreimagesResult, error) {
	payloadResult, err := s.RecoverPayload(ctx, batchNum, batchBlockHash, sequencerMsg)
	if err != nil {
		return nil, err
	}
	return &daprovider.PayloadAndPreimagesResult{
		Payload:   payloadResult.Payload,
		Preimages: make(daprovider.PreimagesMap),
	}, nil
}

// daprovider_generateCertificateValidityProof(certificate)
func (s *Server) GenerateCertificateValidityProof(ctx context.Context, certificate hexutil.Bytes) (*server_api.GenerateCertificateValidityProofResult, error) {
	ok, err := s.store.Exists(ctx, certificate)
	if err != nil {
		return nil, err
	}
	proof := []byte{0x00, 0x01}
	if ok {
		proof[0] = 0x01
	}
	return &server_api.GenerateCertificateValidityProofResult{Proof: proof}, nil
}

// daprovider_generateReadPreimageProof(offset, certificate)
func (s *Server) GenerateReadPreimageProof(ctx context.Context, offset hexutil.Uint64, certificate hexutil.Bytes) (*server_api.GenerateReadPreimageProofResult, error) {
	payload, err := s.store.GetPayloadByCert(ctx, certificate)
	if err != nil {
		return nil, err
	}
	off := uint64(offset)
	if off > uint64(len(payload)) {
		return nil, fmt.Errorf("offset out of bounds")
	}
	end := off + 32
	if end > uint64(len(payload)) {
		end = uint64(len(payload))
	}
	chunk := payload[off:end]
	proof := make([]byte, 0, 1+8+1+len(chunk))
	proof = append(proof, 0x01) // mock read-proof version
	proof = append(proof, byte(off>>56), byte(off>>48), byte(off>>40), byte(off>>32), byte(off>>24), byte(off>>16), byte(off>>8), byte(off))
	proof = append(proof, byte(len(chunk)))
	proof = append(proof, chunk...)
	return &server_api.GenerateReadPreimageProofResult{Proof: proof}, nil
}

func StartRPC(ctx context.Context, addr string, port uint64, srv *Server) (*http.Server, net.Listener, error) {
	if srv == nil {
		return nil, nil, errors.New("nil server")
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return nil, nil, err
	}
	rpcSrv := rpc.NewServer()
	if err := rpcSrv.RegisterName("daprovider", srv); err != nil {
		_ = ln.Close()
		return nil, nil, err
	}
	httpSrv := &http.Server{Handler: rpcSrv}
	go func() { _ = httpSrv.Serve(ln) }()
	go func() {
		<-ctx.Done()
		_ = httpSrv.Shutdown(context.Background())
	}()
	return httpSrv, ln, nil
}
