// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
	"crypto/sha256"
	"math/big"
	"net/http"
	"strings"
	"testing"

	celestiacert "github.com/celestiaorg/nitro-das-celestia/daserver/cert"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tidwall/gjson"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider/celestiada"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	dapserver "github.com/offchainlabs/nitro/daprovider/server"
)

func createCelestiaDAProviderServer(
	t *testing.T,
	ctx context.Context,
	store *celestiada.Store,
	l1Client *ethclient.Client,
	mockBlobstreamAddr common.Address,
	port int,
) (*http.Server, string, *celestiada.Provider) {
	t.Helper()

	provider := celestiada.NewProvider(store, l1Client, mockBlobstreamAddr)
	serverConfig := &dapserver.ServerConfig{
		Addr:               "127.0.0.1",
		Port:               uint64(port),
		EnableDAWriter:     true,
		ServerTimeouts:     genericconf.HTTPServerTimeoutConfigDefault,
		RPCServerBodyLimit: genericconf.HTTPServerBodyLimitDefault,
	}
	server, err := dapserver.NewServerWithDAPProvider(
		ctx,
		serverConfig,
		provider,
		provider,
		provider,
		[]byte{celestiacert.CustomDAHeaderFlag},
		data_streaming.PayloadCommitmentVerifier(),
	)
	Require(t, err)

	t.Logf("Started CelestiaDA provider server at %s", server.Addr)
	return server, server.Addr, provider
}

func createEvilCelestiaDAProviderServer(
	t *testing.T,
	ctx context.Context,
	store *celestiada.Store,
	l1Client *ethclient.Client,
	mockBlobstreamAddr common.Address,
	port int,
) (*http.Server, string, *EvilCelestiaDAProvider) {
	t.Helper()

	baseProvider := celestiada.NewProvider(store, l1Client, mockBlobstreamAddr)
	evilProvider := NewEvilCelestiaDAProvider(&celestiaDAProviderAdapter{provider: baseProvider})
	evilAPI := newEvilCelestiaDAAPI(evilProvider)

	serverConfig := &dapserver.ServerConfig{
		Addr:               "127.0.0.1",
		Port:               uint64(port),
		EnableDAWriter:     true,
		ServerTimeouts:     genericconf.HTTPServerTimeoutConfigDefault,
		RPCServerBodyLimit: genericconf.HTTPServerBodyLimitDefault,
	}
	server, err := dapserver.NewServerWithDAPProvider(
		ctx,
		serverConfig,
		evilAPI,
		baseProvider,
		evilAPI,
		[]byte{celestiacert.CustomDAHeaderFlag},
		data_streaming.PayloadCommitmentVerifier(),
	)
	Require(t, err)

	t.Logf("Started evil CelestiaDA provider server at %s", server.Addr)
	return server, server.Addr, evilProvider
}

func mutateCelestiaCertTxCommitment(certBytes []byte) []byte {
	mutated := append([]byte(nil), certBytes...)
	if len(mutated) >= celestiacert.CelestiaDACertV1Len {
		mutated[28] ^= 0x01
	}
	return mutated
}

func submitMockBlobstreamForCelestiaCert(
	t *testing.T,
	ctx context.Context,
	client *ethclient.Client,
	opts *bind.TransactOpts,
	mockBlobstreamAddr common.Address,
	certBytes []byte,
) {
	t.Helper()

	parsed := &celestiacert.CelestiaDACertV1{}
	Require(t, parsed.UnmarshalBinary(certBytes))

	blobstreamABI := gjson.GetBytes(mockBlobstreamArtifact, "abi").Raw
	parsedABI, err := abi.JSON(strings.NewReader(blobstreamABI))
	Require(t, err)
	bound := bind.NewBoundContract(mockBlobstreamAddr, parsedABI, client, client, client)

	bytes32Ty, err := abi.NewType("bytes32", "", nil)
	Require(t, err)
	commitmentArgs := abi.Arguments{{Type: bytes32Ty}}
	packedCommitment, err := commitmentArgs.Pack(singleBlockTupleRoot(parsed.BlockHeight, parsed.DataRoot))
	Require(t, err)
	var commitment [32]byte
	copy(commitment[:], packedCommitment[:32])

	tx, err := bound.Transact(opts, "submitDataCommitment", commitment, parsed.BlockHeight, parsed.BlockHeight+1)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
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
