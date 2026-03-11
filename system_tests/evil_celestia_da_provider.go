// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	celestiacert "github.com/celestiaorg/nitro-das-celestia/daserver/cert"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/celestiada"
	"github.com/offchainlabs/nitro/util/containers"
)

type celestiaDAReader interface {
	Read(context.Context, *celestiacert.CelestiaDACertV1) (*celestiada.ReadResult, error)
	GenerateReadPreimageProof(context.Context, uint64, []byte) ([]byte, error)
	GenerateCertificateValidityProof(context.Context, []byte) ([]byte, error)
}

type celestiaDAProviderAdapter struct {
	provider *celestiada.Provider
}

func (a *celestiaDAProviderAdapter) Read(ctx context.Context, cert *celestiacert.CelestiaDACertV1) (*celestiada.ReadResult, error) {
	return a.provider.Read(ctx, cert)
}

func (a *celestiaDAProviderAdapter) GenerateReadPreimageProof(ctx context.Context, offset uint64, certificate []byte) ([]byte, error) {
	result, err := a.provider.GenerateReadPreimageProof(offset, certificate).Await(ctx)
	if err != nil {
		return nil, err
	}
	return result.Proof, nil
}

func (a *celestiaDAProviderAdapter) GenerateCertificateValidityProof(ctx context.Context, certificate []byte) ([]byte, error) {
	result, err := a.provider.GenerateCertificateValidityProof(certificate).Await(ctx)
	if err != nil {
		return nil, err
	}
	return result.Proof, nil
}

type EvilCelestiaDAProvider struct {
	reader celestiaDAReader

	mu                   sync.RWMutex
	evilPayloadByCert    map[common.Hash][]byte
	readAliasByCert      map[common.Hash][]byte
	readProofAliasByCert map[common.Hash][]byte
	claimInvalidByCert   map[common.Hash]bool
	claimValidByCert     map[common.Hash]bool
}

func NewEvilCelestiaDAProvider(reader celestiaDAReader) *EvilCelestiaDAProvider {
	return &EvilCelestiaDAProvider{
		reader:               reader,
		evilPayloadByCert:    make(map[common.Hash][]byte),
		readAliasByCert:      make(map[common.Hash][]byte),
		readProofAliasByCert: make(map[common.Hash][]byte),
		claimInvalidByCert:   make(map[common.Hash]bool),
		claimValidByCert:     make(map[common.Hash]bool),
	}
}

func (e *EvilCelestiaDAProvider) SetEvilPayload(certHash common.Hash, payload []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.evilPayloadByCert[certHash] = append([]byte(nil), payload...)
}

func (e *EvilCelestiaDAProvider) SetReadAlias(certHash common.Hash, aliasedCert []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.readAliasByCert[certHash] = append([]byte(nil), aliasedCert...)
}

func (e *EvilCelestiaDAProvider) SetReadPreimageProofAlias(certHash common.Hash, aliasedCert []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.readProofAliasByCert[certHash] = append([]byte(nil), aliasedCert...)
}

func (e *EvilCelestiaDAProvider) SetClaimCertInvalid(certHash common.Hash) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.claimInvalidByCert[certHash] = true
}

func (e *EvilCelestiaDAProvider) SetClaimCertValid(certHash common.Hash) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.claimValidByCert[certHash] = true
}

func celestiaCertHash(certBytes []byte) common.Hash {
	return crypto.Keccak256Hash(certBytes)
}

func celestiaCertHashFromParsed(cert *celestiacert.CelestiaDACertV1) (common.Hash, error) {
	certBytes, err := cert.MarshalBinary()
	if err != nil {
		return common.Hash{}, err
	}
	return celestiaCertHash(certBytes), nil
}

func parseCelestiaCert(certBytes []byte) (*celestiacert.CelestiaDACertV1, error) {
	parsed := &celestiacert.CelestiaDACertV1{}
	if err := parsed.UnmarshalBinary(certBytes); err != nil {
		return nil, err
	}
	return parsed, nil
}

func cloneCelestiaReadResult(res *celestiada.ReadResult) *celestiada.ReadResult {
	if res == nil {
		return nil
	}
	clone := *res
	clone.Message = append([]byte(nil), res.Message...)
	return &clone
}

func (e *EvilCelestiaDAProvider) effectiveReadCert(cert *celestiacert.CelestiaDACertV1) (*celestiacert.CelestiaDACertV1, common.Hash, error) {
	certHash, err := celestiaCertHashFromParsed(cert)
	if err != nil {
		return nil, common.Hash{}, err
	}

	e.mu.RLock()
	aliasedCert := append([]byte(nil), e.readAliasByCert[certHash]...)
	e.mu.RUnlock()
	if len(aliasedCert) == 0 {
		return cert, certHash, nil
	}

	parsed, err := parseCelestiaCert(aliasedCert)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("parse aliased Celestia cert: %w", err)
	}
	return parsed, certHash, nil
}

func (e *EvilCelestiaDAProvider) shouldBypassPreimageValidation(certHash common.Hash) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.evilPayloadByCert[certHash]) != 0 || len(e.readAliasByCert[certHash]) != 0
}

func (e *EvilCelestiaDAProvider) Read(ctx context.Context, cert *celestiacert.CelestiaDACertV1) (*celestiada.ReadResult, error) {
	certHash, err := celestiaCertHashFromParsed(cert)
	if err != nil {
		return nil, err
	}

	e.mu.RLock()
	if e.claimInvalidByCert[certHash] {
		e.mu.RUnlock()
		return nil, fmt.Errorf("certificate validation failed: claimed to be invalid")
	}
	evilPayload := append([]byte(nil), e.evilPayloadByCert[certHash]...)
	e.mu.RUnlock()

	effectiveCert, _, err := e.effectiveReadCert(cert)
	if err != nil {
		return nil, err
	}

	result, err := e.reader.Read(ctx, effectiveCert)
	if err != nil {
		return nil, err
	}
	if len(evilPayload) == 0 {
		return result, nil
	}

	cloned := cloneCelestiaReadResult(result)
	cloned.Message = evilPayload
	return cloned, nil
}

func (e *EvilCelestiaDAProvider) GenerateReadPreimageProof(
	ctx context.Context,
	offset uint64,
	cert *celestiacert.CelestiaDACertV1,
) ([]byte, error) {
	certHash, err := celestiaCertHashFromParsed(cert)
	if err != nil {
		return nil, err
	}

	e.mu.RLock()
	aliasedCert := append([]byte(nil), e.readProofAliasByCert[certHash]...)
	e.mu.RUnlock()
	if len(aliasedCert) == 0 {
		certBytes, err := cert.MarshalBinary()
		if err != nil {
			return nil, err
		}
		return e.reader.GenerateReadPreimageProof(ctx, offset, certBytes)
	}

	parsed, err := parseCelestiaCert(aliasedCert)
	if err != nil {
		return nil, fmt.Errorf("parse aliased read-preimage cert: %w", err)
	}
	certBytes, err := parsed.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return e.reader.GenerateReadPreimageProof(ctx, offset, certBytes)
}

func (e *EvilCelestiaDAProvider) GenerateCertificateValidityProof(
	ctx context.Context,
	cert *celestiacert.CelestiaDACertV1,
) ([]byte, error) {
	certHash, err := celestiaCertHashFromParsed(cert)
	if err != nil {
		return nil, err
	}

	e.mu.RLock()
	claimValid := e.claimValidByCert[certHash]
	claimInvalid := e.claimInvalidByCert[certHash]
	e.mu.RUnlock()

	certBytes, err := cert.MarshalBinary()
	if err != nil {
		return nil, err
	}
	proof, err := e.reader.GenerateCertificateValidityProof(ctx, certBytes)
	if err != nil {
		return nil, err
	}
	if len(proof) == 0 {
		return nil, fmt.Errorf("missing Celestia validity proof")
	}

	switch {
	case claimValid:
		proof[0] = ValidityProofValid
	case claimInvalid:
		proof[0] = ValidityProofInvalid
	}
	return proof, nil
}

type evilCelestiaDAAPI struct {
	provider   *EvilCelestiaDAProvider
	baseReader daprovider.Reader
}

func newEvilCelestiaDAAPI(provider *EvilCelestiaDAProvider) *evilCelestiaDAAPI {
	return &evilCelestiaDAAPI{
		provider:   provider,
		baseReader: celestiada.NewReader(provider),
	}
}

func (e *evilCelestiaDAAPI) recoverInternal(
	ctx context.Context,
	sequencerMsg []byte,
	needPreimages bool,
) ([]byte, daprovider.PreimagesMap, error) {
	certificate, err := celestiacert.ExtractFromSequencerMessage(sequencerMsg)
	if err != nil {
		return nil, nil, err
	}
	certHash, err := celestiaCertHashFromParsed(certificate)
	if err != nil {
		return nil, nil, err
	}
	if !e.provider.shouldBypassPreimageValidation(certHash) {
		return nil, nil, fmt.Errorf("evil path bypass not needed for cert %s", certHash.Hex())
	}

	result, err := e.provider.Read(ctx, certificate)
	if err != nil {
		return nil, nil, err
	}
	if len(result.Message) == 0 {
		return nil, nil, errors.New("empty payload returned from Celestia")
	}
	if !needPreimages {
		return result.Message, nil, nil
	}

	preimages := make(daprovider.PreimagesMap)
	preimageRecorder := daprovider.RecordPreimagesTo(preimages)
	certBytes, err := certificate.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	preimageRecorder(crypto.Keccak256Hash(certBytes), result.Message, arbutil.DACertificatePreimageType)

	return result.Message, preimages, nil
}

func (e *evilCelestiaDAAPI) RecoverPayload(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadResult] {
	certificate, err := celestiacert.ExtractFromSequencerMessage(sequencerMsg)
	if err == nil {
		certHash, hashErr := celestiaCertHashFromParsed(certificate)
		if hashErr == nil && e.provider.shouldBypassPreimageValidation(certHash) {
			return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadResult, error) {
				payload, _, err := e.recoverInternal(ctx, sequencerMsg, false)
				return daprovider.PayloadResult{Payload: payload}, err
			})
		}
	}
	return e.baseReader.RecoverPayload(batchNum, batchBlockHash, sequencerMsg)
}

func (e *evilCelestiaDAAPI) CollectPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PreimagesResult] {
	certificate, err := celestiacert.ExtractFromSequencerMessage(sequencerMsg)
	if err == nil {
		certHash, hashErr := celestiaCertHashFromParsed(certificate)
		if hashErr == nil && e.provider.shouldBypassPreimageValidation(certHash) {
			return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimagesResult, error) {
				_, preimages, err := e.recoverInternal(ctx, sequencerMsg, true)
				return daprovider.PreimagesResult{Preimages: preimages}, err
			})
		}
	}
	return e.baseReader.CollectPreimages(batchNum, batchBlockHash, sequencerMsg)
}

func (e *evilCelestiaDAAPI) RecoverPayloadAndPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadAndPreimagesResult] {
	certificate, err := celestiacert.ExtractFromSequencerMessage(sequencerMsg)
	if err == nil {
		certHash, hashErr := celestiaCertHashFromParsed(certificate)
		if hashErr == nil && e.provider.shouldBypassPreimageValidation(certHash) {
			return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadAndPreimagesResult, error) {
				payload, preimages, err := e.recoverInternal(ctx, sequencerMsg, true)
				return daprovider.PayloadAndPreimagesResult{
					Payload:   payload,
					Preimages: preimages,
				}, err
			})
		}
	}
	return e.baseReader.RecoverPayloadAndPreimages(batchNum, batchBlockHash, sequencerMsg)
}

func (e *evilCelestiaDAAPI) GenerateReadPreimageProof(
	offset uint64,
	certificate []byte,
) containers.PromiseInterface[daprovider.PreimageProofResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimageProofResult, error) {
		parsed, err := parseCelestiaCert(certificate)
		if err != nil {
			return daprovider.PreimageProofResult{}, err
		}
		proof, err := e.provider.GenerateReadPreimageProof(ctx, offset, parsed)
		return daprovider.PreimageProofResult{Proof: proof}, err
	})
}

func (e *evilCelestiaDAAPI) GenerateCertificateValidityProof(
	certificate []byte,
) containers.PromiseInterface[daprovider.ValidityProofResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.ValidityProofResult, error) {
		parsed, err := parseCelestiaCert(certificate)
		if err != nil {
			return daprovider.ValidityProofResult{}, err
		}
		proof, err := e.provider.GenerateCertificateValidityProof(ctx, parsed)
		return daprovider.ValidityProofResult{Proof: proof}, err
	})
}
