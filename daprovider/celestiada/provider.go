// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package celestiada

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"

	celestiaappda "github.com/celestiaorg/celestia-app/v6/pkg/da"
	celestiaappproof "github.com/celestiaorg/celestia-app/v6/pkg/proof"
	square "github.com/celestiaorg/go-square/v3"
	libshare "github.com/celestiaorg/go-square/v3/share"
	celestiacert "github.com/celestiaorg/nitro-das-celestia/daserver/cert"
	"github.com/celestiaorg/rsmt2d"
	"github.com/cometbft/cometbft/crypto/merkle"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	blobstreamx "github.com/succinctlabs/sp1-blobstream/bindings"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/containers"
)

const (
	DefaultMaxMessageSize      = 32 * 1024 * 1024
	DefaultNamespaceHex        = "0000008e5f679bf7116c"
	celestiaFirstShareDataCap  = 478
	celestiaContShareDataCap   = 482
	celestiaReadProofVersion   = 0x01
	celestiaValidityProofValid = 0x01
)

type ReadResult struct {
	Message     []byte
	RowRoots    [][]byte
	ColumnRoots [][]byte
	Rows        [][][]byte
	SquareSize  uint64
	StartRow    uint64
	EndRow      uint64
}

type record struct {
	certificate []byte
	cert        *celestiacert.CelestiaDACertV1
	payload     []byte
	eds         *rsmt2d.ExtendedDataSquare
	rowRoots    [][]byte
	columnRoots [][]byte
}

type Store struct {
	mu         sync.RWMutex
	nextHeight uint64
	namespace  libshare.Namespace
	byCert     map[string]*record
}

func NewStore() *Store {
	nsBytes, err := hex.DecodeString(DefaultNamespaceHex)
	if err != nil {
		panic(fmt.Sprintf("invalid celestia test namespace: %v", err))
	}
	namespace, err := libshare.NewV0Namespace(nsBytes)
	if err != nil {
		panic(fmt.Sprintf("invalid celestia test namespace: %v", err))
	}
	return &Store{
		nextHeight: 1,
		namespace:  namespace,
		byCert:     make(map[string]*record),
	}
}

func (s *Store) Namespace() libshare.Namespace {
	return s.namespace
}

func (s *Store) Put(_ context.Context, payload []byte) ([]byte, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload cannot be empty")
	}

	blob, err := libshare.NewV0Blob(s.namespace, payload)
	if err != nil {
		return nil, err
	}
	shares, err := blob.ToShares()
	if err != nil {
		return nil, err
	}

	odsSize := nextSquareSize(len(shares))
	padding := odsSize*odsSize - len(shares)
	flatShares := append([]libshare.Share(nil), shares...)
	flatShares = append(flatShares, libshare.TailPaddingShares(padding)...)
	ods := square.Square(flatShares)

	eds, err := celestiaappda.ExtendShares(libshare.ToBytes(ods))
	if err != nil {
		return nil, err
	}
	rowRoots, err := eds.RowRoots()
	if err != nil {
		return nil, err
	}
	columnRoots, err := eds.ColRoots()
	if err != nil {
		return nil, err
	}

	roots := make([][]byte, 0, len(rowRoots)+len(columnRoots))
	roots = append(roots, rowRoots...)
	roots = append(roots, columnRoots...)
	dataRootBytes := merkle.HashFromByteSlices(roots)
	var dataRoot [32]byte
	copy(dataRoot[:], dataRootBytes)

	txCommitment := sha256.Sum256(payload)

	s.mu.Lock()
	defer s.mu.Unlock()

	height := s.nextHeight
	s.nextHeight++

	cert := celestiacert.NewCelestiaCertificate(
		height,
		0,
		uint64(len(shares)),
		txCommitment,
		dataRoot,
	)
	certBytes, err := cert.MarshalBinary()
	if err != nil {
		return nil, err
	}

	s.byCert[string(certBytes)] = &record{
		certificate: append([]byte(nil), certBytes...),
		cert:        cert,
		payload:     append([]byte(nil), payload...),
		eds:         eds,
		rowRoots:    clone2D(rowRoots),
		columnRoots: clone2D(columnRoots),
	}
	return certBytes, nil
}

func (s *Store) lookup(certificate []byte) (*record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.byCert[string(certificate)]
	return rec, ok
}

type Provider struct {
	store          *Store
	l1Client       *ethclient.Client
	blobstreamAddr common.Address
	maxMessageSize int
}

func NewProvider(store *Store, l1Client *ethclient.Client, blobstreamAddr common.Address) *Provider {
	if store == nil {
		store = NewStore()
	}
	return &Provider{
		store:          store,
		l1Client:       l1Client,
		blobstreamAddr: blobstreamAddr,
		maxMessageSize: DefaultMaxMessageSize,
	}
}

func (p *Provider) SetMaxMessageSize(max int) {
	if max > 0 {
		p.maxMessageSize = max
	}
}

func (p *Provider) Store(message []byte, _ uint64) containers.PromiseInterface[[]byte] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) ([]byte, error) {
		if len(message) > p.maxMessageSize {
			return nil, daprovider.ErrMessageTooLarge
		}
		return p.store.Put(ctx, message)
	})
}

func (p *Provider) GetMaxMessageSize() containers.PromiseInterface[int] {
	return containers.DoPromise(context.Background(), func(context.Context) (int, error) {
		return p.maxMessageSize, nil
	})
}

func (p *Provider) Read(ctx context.Context, cert *celestiacert.CelestiaDACertV1) (*ReadResult, error) {
	if cert == nil {
		return nil, certificateValidationError("missing certificate")
	}
	certBytes, err := cert.MarshalBinary()
	if err != nil {
		return nil, certificateValidationError("malformed certificate")
	}
	rec, ok := p.store.lookup(certBytes)
	if !ok {
		return nil, certificateValidationError("unknown certificate")
	}

	squareSize := uint64(len(rec.rowRoots))
	odsSize := squareSize / 2
	if odsSize == 0 {
		return nil, certificateValidationError("invalid square size")
	}
	if cert.SharesLength == 0 {
		return nil, certificateValidationError("share range underflow")
	}
	if cert.Start > math.MaxUint64-(cert.SharesLength-1) {
		return nil, certificateValidationError("share range overflow")
	}
	endIndex := cert.Start + cert.SharesLength - 1
	if endIndex >= odsSize*odsSize {
		return nil, certificateValidationError("share range out of bounds")
	}
	startRow := cert.Start / odsSize
	endRow := endIndex / odsSize

	rows := make([][][]byte, 0, endRow-startRow+1)
	for row := startRow; row <= endRow; row++ {
		rows = append(rows, clone2D(rec.eds.Row(uint(row))))
	}
	return &ReadResult{
		Message:     append([]byte(nil), rec.payload...),
		RowRoots:    clone2D(rec.rowRoots),
		ColumnRoots: clone2D(rec.columnRoots),
		Rows:        rows,
		SquareSize:  squareSize,
		StartRow:    startRow,
		EndRow:      endRow,
	}, nil
}

func (p *Provider) RecoverPayload(_ uint64, _ common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PayloadResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadResult, error) {
		cert, err := celestiacert.ExtractFromSequencerMessage(sequencerMsg)
		if err != nil {
			return daprovider.PayloadResult{}, err
		}
		result, err := p.Read(ctx, cert)
		if err != nil {
			return daprovider.PayloadResult{}, err
		}
		return daprovider.PayloadResult{Payload: result.Message}, nil
	})
}

func (p *Provider) CollectPreimages(_ uint64, _ common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimagesResult, error) {
		cert, err := celestiacert.ExtractFromSequencerMessage(sequencerMsg)
		if err != nil {
			return daprovider.PreimagesResult{}, err
		}
		result, err := p.Read(ctx, cert)
		if err != nil {
			return daprovider.PreimagesResult{}, err
		}
		certBytes, err := cert.MarshalBinary()
		if err != nil {
			return daprovider.PreimagesResult{}, err
		}
		preimages := make(daprovider.PreimagesMap)
		daprovider.RecordPreimagesTo(preimages)(
			crypto.Keccak256Hash(certBytes),
			result.Message,
			arbutil.DACertificatePreimageType,
		)
		return daprovider.PreimagesResult{Preimages: preimages}, nil
	})
}

func (p *Provider) RecoverPayloadAndPreimages(batchNum uint64, batchBlockHash common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PayloadAndPreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadAndPreimagesResult, error) {
		payload, err := p.RecoverPayload(batchNum, batchBlockHash, sequencerMsg).Await(ctx)
		if err != nil {
			return daprovider.PayloadAndPreimagesResult{}, err
		}
		preimages, err := p.CollectPreimages(batchNum, batchBlockHash, sequencerMsg).Await(ctx)
		if err != nil {
			return daprovider.PayloadAndPreimagesResult{}, err
		}
		return daprovider.PayloadAndPreimagesResult{
			Payload:   payload.Payload,
			Preimages: preimages.Preimages,
		}, nil
	})
}

func (p *Provider) GenerateCertificateValidityProof(certificate []byte) containers.PromiseInterface[daprovider.ValidityProofResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.ValidityProofResult, error) {
		proof := []byte{0x00, celestiaValidityProofValid}

		cert := &celestiacert.CelestiaDACertV1{}
		if err := cert.UnmarshalBinary(certificate); err != nil {
			return daprovider.ValidityProofResult{Proof: proof}, nil
		}
		if cert.SharesLength == 0 || cert.DataRoot == ([32]byte{}) {
			return daprovider.ValidityProofResult{Proof: proof}, nil
		}

		attestationProof, err := p.generateCertificateAttestationProof(ctx, cert)
		if err != nil {
			return daprovider.ValidityProofResult{}, err
		}
		if attestationProof == nil {
			return daprovider.ValidityProofResult{Proof: proof}, nil
		}

		proofData, err := packValidityProof(*attestationProof)
		if err != nil {
			return daprovider.ValidityProofResult{}, err
		}
		proof[0] = 0x01
		proof = append(proof, proofData...)
		return daprovider.ValidityProofResult{Proof: proof}, nil
	})
}

func (p *Provider) GenerateReadPreimageProof(offset uint64, certificate []byte) containers.PromiseInterface[daprovider.PreimageProofResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimageProofResult, error) {
		cert := &celestiacert.CelestiaDACertV1{}
		if err := cert.UnmarshalBinary(certificate); err != nil {
			return daprovider.PreimageProofResult{}, err
		}
		rec, ok := p.store.lookup(certificate)
		if !ok {
			return daprovider.PreimageProofResult{}, certificateValidationError("unknown certificate")
		}

		plan := buildReadChunkPlan(offset, uint64(len(rec.payload)), cert.Start)
		shareProof, err := celestiaappproof.NewShareInclusionProofFromEDS(
			rec.eds,
			p.store.namespace,
			libshare.NewRange(int(plan.firstShareIndex), int(plan.firstShareIndex+plan.shareCount)),
		)
		if err != nil {
			return daprovider.PreimageProofResult{}, err
		}

		event, found, err := p.findCommitmentEvent(ctx, cert.BlockHeight)
		if err != nil {
			return daprovider.PreimageProofResult{}, err
		}
		if !found {
			return daprovider.PreimageProofResult{}, certificateValidationError("missing blobstream commitment")
		}
		if event.EndBlock-event.StartBlock != 1 || event.StartBlock != cert.BlockHeight {
			return daprovider.PreimageProofResult{}, fmt.Errorf("unsupported blobstream commitment range [%d,%d) for height %d", event.StartBlock, event.EndBlock, cert.BlockHeight)
		}

		proofData, err := packSharesProof(p.blobstreamAddr, shareProofToABIProof(shareProof, cert, event.ProofNonce.Uint64()))
		if err != nil {
			return daprovider.PreimageProofResult{}, err
		}

		return daprovider.PreimageProofResult{
			Proof: buildReadPreimageProof(offset, uint64(len(rec.payload)), plan, proofData),
		}, nil
	})
}

func (p *Provider) findCommitmentEvent(ctx context.Context, height uint64) (*blobstreamx.BindingsDataCommitmentStored, bool, error) {
	if p.l1Client == nil {
		return nil, false, errors.New("missing L1 client")
	}
	if p.blobstreamAddr == (common.Address{}) {
		return nil, false, errors.New("missing blobstream address")
	}

	binding, err := blobstreamx.NewBindings(p.blobstreamAddr, p.l1Client)
	if err != nil {
		return nil, false, err
	}
	latestBlock, err := p.l1Client.BlockNumber(ctx)
	if err != nil {
		return nil, false, err
	}
	iter, err := binding.FilterDataCommitmentStored(
		&bind.FilterOpts{Start: 0, End: &latestBlock, Context: ctx},
		nil,
		nil,
		nil,
	)
	if err != nil {
		return nil, false, err
	}
	defer iter.Close()

	var match *blobstreamx.BindingsDataCommitmentStored
	for iter.Next() {
		event := iter.Event
		if event.StartBlock <= height && height < event.EndBlock {
			match = &blobstreamx.BindingsDataCommitmentStored{
				ProofNonce:     event.ProofNonce,
				StartBlock:     event.StartBlock,
				EndBlock:       event.EndBlock,
				DataCommitment: event.DataCommitment,
			}
		}
	}
	if err := iter.Error(); err != nil {
		return nil, false, err
	}
	if match == nil {
		return nil, false, nil
	}
	return match, true, nil
}

type readChunkPlan struct {
	chunkLen        uint8
	firstShareIndex uint64
	shareCount      uint64
}

func buildReadChunkPlan(offset, payloadSize, certStart uint64) readChunkPlan {
	shareRel := payloadOffsetToShareRel(offset)
	plan := readChunkPlan{
		chunkLen:        expectedChunkLen(offset, payloadSize),
		firstShareIndex: certStart + shareRel,
		shareCount:      1,
	}
	if plan.chunkLen == 0 {
		return plan
	}

	sharePayloadStart := payloadStartForShareRel(shareRel)
	sharePayloadCap := payloadCapacityForShareRel(shareRel)
	intra := offset - sharePayloadStart
	if intra+uint64(plan.chunkLen) > sharePayloadCap {
		plan.shareCount = 2
	}
	return plan
}

func buildReadPreimageProof(offset, payloadSize uint64, plan readChunkPlan, proofData []byte) []byte {
	proof := make([]byte, 0, 1+8+8+1+8+1+len(proofData))
	proof = append(proof, celestiaReadProofVersion)
	proof = binary.BigEndian.AppendUint64(proof, offset)
	proof = binary.BigEndian.AppendUint64(proof, payloadSize)
	proof = append(proof, plan.chunkLen)
	proof = binary.BigEndian.AppendUint64(proof, plan.firstShareIndex)
	proof = append(proof, byte(plan.shareCount))
	proof = append(proof, proofData...)
	return proof
}

func payloadOffsetToShareRel(offset uint64) uint64 {
	if offset < celestiaFirstShareDataCap {
		return 0
	}
	return 1 + (offset-celestiaFirstShareDataCap)/celestiaContShareDataCap
}

func payloadStartForShareRel(shareRel uint64) uint64 {
	if shareRel == 0 {
		return 0
	}
	return celestiaFirstShareDataCap + (shareRel-1)*celestiaContShareDataCap
}

func payloadCapacityForShareRel(shareRel uint64) uint64 {
	if shareRel == 0 {
		return celestiaFirstShareDataCap
	}
	return celestiaContShareDataCap
}

func expectedChunkLen(offset, payloadSize uint64) uint8 {
	if offset >= payloadSize {
		return 0
	}
	remaining := payloadSize - offset
	if remaining > 32 {
		return 32
	}
	return uint8(remaining)
}

func certificateValidationError(reason string) error {
	return &daprovider.CertificateValidationError{Reason: fmt.Sprintf("certificate validation failed: %s", reason)}
}

func nextSquareSize(shareCount int) int {
	size := 1
	for size*size < shareCount {
		size <<= 1
	}
	return size
}

func clone2D(in [][]byte) [][]byte {
	out := make([][]byte, len(in))
	for i := range in {
		out[i] = append([]byte(nil), in[i]...)
	}
	return out
}

func shareProofToABIProof(
	shareProof celestiaappproof.ShareProof,
	cert *celestiacert.CelestiaDACertV1,
	proofNonce uint64,
) SharesProof {
	result := SharesProof{
		Data:        clone2D(shareProof.Data),
		ShareProofs: make([]NamespaceMerkleMultiproof, 0, len(shareProof.ShareProofs)),
		Namespace: Namespace{
			Version: [1]byte{byte(shareProof.NamespaceVersion)},
			Id:      namespaceID28(shareProof.NamespaceId),
		},
		RowRoots:  make([]NamespaceNode, 0, len(shareProof.RowProof.RowRoots)),
		RowProofs: make([]BinaryMerkleProof, 0, len(shareProof.RowProof.Proofs)),
		AttestationProof: AttestationProof{
			TupleRootNonce: uint256Big(proofNonce),
			Tuple: DataRootTuple{
				Height:   uint256Big(cert.BlockHeight),
				DataRoot: cert.DataRoot,
			},
			Proof: BinaryMerkleProof{
				SideNodes: nil,
				Key:       uint256Big(0),
				NumLeaves: uint256Big(1),
			},
		},
	}

	for _, sp := range shareProof.ShareProofs {
		sideNodes := make([]NamespaceNode, 0, len(sp.Nodes))
		for _, nodeBytes := range sp.Nodes {
			sideNodes = append(sideNodes, toNamespaceNode(nodeBytes))
		}
		result.ShareProofs = append(result.ShareProofs, NamespaceMerkleMultiproof{
			BeginKey:  uint256Big(uint64(sp.Start)),
			EndKey:    uint256Big(uint64(sp.End)),
			SideNodes: sideNodes,
		})
	}
	for _, rowRoot := range shareProof.RowProof.RowRoots {
		result.RowRoots = append(result.RowRoots, toNamespaceNode(rowRoot))
	}
	for _, rowProof := range shareProof.RowProof.Proofs {
		result.RowProofs = append(result.RowProofs, toRowProof(rowProof))
	}
	return result
}

func (p *Provider) generateCertificateAttestationProof(
	ctx context.Context,
	cert *celestiacert.CelestiaDACertV1,
) (*AttestationProof, error) {
	event, found, err := p.findCommitmentEvent(ctx, cert.BlockHeight)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	if event.EndBlock-event.StartBlock != 1 || event.StartBlock != cert.BlockHeight {
		return nil, nil
	}
	if event.DataCommitment != singleBlockTupleRoot(cert.BlockHeight, cert.DataRoot) {
		return nil, nil
	}

	attestationProof := AttestationProof{
		TupleRootNonce: uint256Big(event.ProofNonce.Uint64()),
		Tuple: DataRootTuple{
			Height:   uint256Big(cert.BlockHeight),
			DataRoot: cert.DataRoot,
		},
		Proof: BinaryMerkleProof{
			SideNodes: nil,
			Key:       uint256Big(0),
			NumLeaves: uint256Big(1),
		},
	}
	return &attestationProof, nil
}

func namespaceID28(in []byte) [28]byte {
	var out [28]byte
	copy(out[:], in)
	return out
}

func uint256Big(v uint64) *big.Int {
	return new(big.Int).SetUint64(v)
}

func singleBlockTupleRoot(height uint64, dataRoot [32]byte) [32]byte {
	var encodedHeight [32]byte
	new(big.Int).SetUint64(height).FillBytes(encodedHeight[:])

	leafInput := make([]byte, 0, 1+len(encodedHeight)+len(dataRoot))
	leafInput = append(leafInput, 0x00)
	leafInput = append(leafInput, encodedHeight[:]...)
	leafInput = append(leafInput, dataRoot[:]...)
	return sha256.Sum256(leafInput)
}
