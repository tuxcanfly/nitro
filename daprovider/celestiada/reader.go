// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package celestiada

import (
	"context"

	celestiacert "github.com/celestiaorg/nitro-das-celestia/daserver/cert"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/containers"
)

type DAReader interface {
	Read(context.Context, *celestiacert.CelestiaDACertV1) (*ReadResult, error)
}

func NewReader(reader DAReader) daprovider.Reader {
	return &readerAdapter{reader: reader}
}

type readerAdapter struct {
	reader DAReader
}

func (r *readerAdapter) RecoverPayload(_ uint64, _ common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PayloadResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadResult, error) {
		certificate, err := celestiacert.ExtractFromSequencerMessage(sequencerMsg)
		if err != nil {
			return daprovider.PayloadResult{}, err
		}
		result, err := r.reader.Read(ctx, certificate)
		if err != nil {
			return daprovider.PayloadResult{}, err
		}
		return daprovider.PayloadResult{Payload: result.Message}, nil
	})
}

func (r *readerAdapter) CollectPreimages(_ uint64, _ common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimagesResult, error) {
		certificate, err := celestiacert.ExtractFromSequencerMessage(sequencerMsg)
		if err != nil {
			return daprovider.PreimagesResult{}, err
		}
		result, err := r.reader.Read(ctx, certificate)
		if err != nil {
			return daprovider.PreimagesResult{}, err
		}
		certBytes, err := certificate.MarshalBinary()
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

func (r *readerAdapter) RecoverPayloadAndPreimages(batchNum uint64, batchBlockHash common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PayloadAndPreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadAndPreimagesResult, error) {
		payload, err := r.RecoverPayload(batchNum, batchBlockHash, sequencerMsg).Await(ctx)
		if err != nil {
			return daprovider.PayloadAndPreimagesResult{}, err
		}
		preimages, err := r.CollectPreimages(batchNum, batchBlockHash, sequencerMsg).Await(ctx)
		if err != nil {
			return daprovider.PayloadAndPreimagesResult{}, err
		}
		return daprovider.PayloadAndPreimagesResult{
			Payload:   payload.Payload,
			Preimages: preimages.Preimages,
		}, nil
	})
}
