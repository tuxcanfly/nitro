// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	celestiadas "github.com/celestiaorg/nitro-das-celestia/daserver"
	celestiacert "github.com/celestiaorg/nitro-das-celestia/daserver/cert"
	celestiatypes "github.com/celestiaorg/nitro-das-celestia/daserver/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tidwall/gjson"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	dapserver "github.com/offchainlabs/nitro/daprovider/server"
)

const mockBlobstreamTupleRootRangeSize uint64 = 10

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func mustEnv(t *testing.T, name string) string {
	t.Helper()

	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Fatalf("%s must be set for Celestia tests", name)
	}
	return value
}

func newCelestiaDATestConfig(
	t *testing.T,
	l1RPC string,
	l1Client *ethclient.Client,
	mockBlobstreamAddr common.Address,
	validatorAddr common.Address,
) *celestiadas.DAConfig {
	t.Helper()

	celestiaRPC := strings.TrimSpace(os.Getenv("CELESTIA_RPC"))
	if celestiaRPC == "" {
		t.Skipf("CELESTIA_RPC must be set for Celestia tests")
	}
	authToken := strings.TrimSpace(os.Getenv("CELESTIA_AUTH_TOKEN"))
	if authToken == "" {
		t.Skipf("CELESTIA_AUTH_TOKEN must be set for Celestia tests")
	}
	readRPC := envOr("CELESTIA_READ_RPC", celestiaRPC)
	readAuthToken := envOr("CELESTIA_READ_AUTH_TOKEN", authToken)

	return &celestiadas.DAConfig{
		WithWriter:       true,
		Rpc:              celestiaRPC,
		ReadRpc:          readRPC,
		NamespaceId:      envOr("CELESTIA_NAMESPACE", "0000008e5f679bf7116c"),
		AuthToken:        authToken,
		ReadAuthToken:    readAuthToken,
		CacheCleanupTime: time.Minute,
		ValidatorConfig: celestiadas.ValidatorConfig{
			EthClient:          l1RPC,
			BlobstreamAddr:     mockBlobstreamAddr.Hex(),
			ProofValidatorAddr: validatorAddr.Hex(),
			SleepTime:          1,
		},
		L1ClientOverride: l1Client,
		RetryConfig:      celestiadas.DefaultCelestiaRetryConfig,
	}
}

func createEvilCelestiaDAProviderServer(
	t *testing.T,
	ctx context.Context,
	port int,
	cfg *celestiadas.DAConfig,
) (*http.Server, string, *EvilCelestiaDAProvider) {
	t.Helper()

	provider, err := celestiadas.NewCelestiaDA(cfg)
	Require(t, err)
	t.Cleanup(func() {
		_ = provider.Stop()
	})

	evilProvider := NewEvilCelestiaDAProvider(provider)
	serverCfg := &dapserver.ServerConfig{
		Addr:               "127.0.0.1",
		Port:               uint64(port),
		EnableDAWriter:     true,
		ServerTimeouts:     genericconf.HTTPServerTimeoutConfigDefault,
		RPCServerBodyLimit: genericconf.HTTPServerBodyLimitDefault,
	}
	evilAPI := newEvilCelestiaDAAPI(evilProvider)
	server, err := dapserver.NewServerWithDAPProvider(
		ctx,
		serverCfg,
		evilAPI,
		celestiatypes.NewWriterForCelestia(provider),
		evilAPI,
		[]byte{celestiacert.CustomDAHeaderFlag},
		data_streaming.PayloadCommitmentVerifier(),
	)
	Require(t, err)

	serverURL := server.Addr
	t.Logf("Started evil Celestia DA provider server at %s", serverURL)
	return server, serverURL, evilProvider
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

	latestBlock, err := mockBlobstreamLatestBlock(ctx, bound)
	Require(t, err)
	if latestBlock > parsed.BlockHeight {
		t.Fatalf("mock blobstream latestBlock %d is ahead of cert height %d", latestBlock, parsed.BlockHeight)
	}

	rangeStart := parsed.BlockHeight + 1 - minUint64(mockBlobstreamTupleRootRangeSize, parsed.BlockHeight+1)
	if rangeStart < latestBlock {
		rangeStart = latestBlock
	}
	rangeEnd := rangeStart + mockBlobstreamTupleRootRangeSize
	if rangeEnd <= parsed.BlockHeight {
		rangeEnd = parsed.BlockHeight + 1
	}

	// Celestia's tuple-inclusion proof path is unreliable for single-block mock commitments,
	// so wait for enough head progress to commit a real multi-block tuple root.
	tupleRootHex := waitForCelestiaTupleRoot(t, ctx, rangeStart, rangeEnd)

	bytes32Ty, err := abi.NewType("bytes32", "", nil)
	Require(t, err)
	commitmentArgs := abi.Arguments{{Type: bytes32Ty}}
	packedCommitment, err := commitmentArgs.Pack(common.HexToHash(tupleRootHex))
	Require(t, err)
	var commitment [32]byte
	copy(commitment[:], packedCommitment[:32])

	tx, err := bound.Transact(opts, "submitDataCommitment", commitment, rangeStart, rangeEnd)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
}

func mockBlobstreamLatestBlock(ctx context.Context, bound *bind.BoundContract) (uint64, error) {
	var out []any
	if err := bound.Call(&bind.CallOpts{Context: ctx}, &out, "latestBlock"); err != nil {
		return 0, err
	}
	if len(out) != 1 {
		return 0, fmt.Errorf("unexpected latestBlock return values: %d", len(out))
	}
	switch value := out[0].(type) {
	case uint64:
		return value, nil
	case *big.Int:
		return value.Uint64(), nil
	case uint32:
		return uint64(value), nil
	case uint16:
		return uint64(value), nil
	case uint8:
		return uint64(value), nil
	default:
		return 0, fmt.Errorf("unexpected latestBlock type %T", value)
	}
}

func waitForCelestiaTupleRoot(t *testing.T, ctx context.Context, start uint64, end uint64) string {
	t.Helper()

	readRPC := strings.TrimSpace(envOr("CELESTIA_READ_RPC", os.Getenv("CELESTIA_RPC")))
	if readRPC == "" {
		t.Skip("CELESTIA_READ_RPC or CELESTIA_RPC must be set for Celestia mock blobstream submission")
	}
	readAuthToken := strings.TrimSpace(envOr("CELESTIA_READ_AUTH_TOKEN", os.Getenv("CELESTIA_AUTH_TOKEN")))
	if readAuthToken == "" {
		t.Skip("CELESTIA_READ_AUTH_TOKEN or CELESTIA_AUTH_TOKEN must be set for Celestia mock blobstream submission")
	}

	deadline := time.After(2 * time.Minute)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		head, err := celestiaLocalHead(ctx, readRPC, readAuthToken)
		if err == nil && head+1 >= end {
			tupleRoot, err := celestiaTupleRoot(ctx, readRPC, readAuthToken, start, end)
			if err == nil {
				return tupleRoot
			}
			t.Logf("waiting for celestia tuple root [%d,%d): %v", start, end, err)
		} else if err != nil {
			t.Logf("waiting for celestia head >= %d: %v", end-1, err)
		}

		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled while waiting for Celestia tuple root [%d,%d)", start, end)
		case <-deadline:
			t.Fatalf("timed out waiting for Celestia tuple root [%d,%d)", start, end)
		case <-ticker.C:
		}
	}
}

func celestiaLocalHead(ctx context.Context, rpcURL, authToken string) (uint64, error) {
	var result struct {
		Header struct {
			Height string `json:"height"`
		} `json:"header"`
	}
	if err := callCelestiaJSONRPC(ctx, rpcURL, authToken, "header.LocalHead", []any{}, &result); err != nil {
		return 0, err
	}
	return strconv.ParseUint(result.Header.Height, 10, 64)
}

func celestiaTupleRoot(ctx context.Context, rpcURL, authToken string, start, end uint64) (string, error) {
	var result string
	if err := callCelestiaJSONRPC(ctx, rpcURL, authToken, "blobstream.GetDataRootTupleRoot", []any{start, end}, &result); err != nil {
		return "", err
	}
	if !strings.HasPrefix(result, "0x") {
		result = "0x" + result
	}
	return result, nil
}

func callCelestiaJSONRPC(ctx context.Context, rpcURL, authToken, method string, params []any, result any) error {
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Error != nil {
		return fmt.Errorf("%s", envelope.Error.Message)
	}
	return json.Unmarshal(envelope.Result, result)
}

func minUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
