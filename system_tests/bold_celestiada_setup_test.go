// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/celestiada"
	"github.com/offchainlabs/nitro/daprovider/daclient"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/execution_consensus"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util"
)

func TestBOLDCelestiaDA_SetupVerification(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei)
	l2chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	ownerBal := big.NewInt(params.Ether)
	ownerBal.Mul(ownerBal, big.NewInt(1_000_000))
	l2info.GenerateGenesisAccount("Owner", ownerBal)
	l2info.GenerateAccount("Destination")

	sconf := setup.RollupStackConfig{
		UseBlobs:               true,
		UseMockBridge:          false,
		UseMockOneStepProver:   false,
		MinimumAssertionPeriod: 0,
	}

	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.DA.ExternalProvider.Enable = true
	nodeConfig.DA.ExternalProvider.WithWriter = true

	l1info, l1backend, l1client, l1stack, addresses, stakeTokenAddr, asserterOpts, signerCfg := setupL1ForBoldProtocol(
		t, ctx, sconf, l2info, false, nodeConfig, l2chainConfig, true, true,
	)
	defer requireClose(t, l1stack)

	mockBlobstreamAddr := l1info.GetAddress("MockBlobstream")
	validatorAddr := l1info.GetAddress("CelestiaDAProofValidator")
	if mockBlobstreamAddr == (common.Address{}) {
		t.Fatal("mock blobstream address was zero")
	}
	if validatorAddr == (common.Address{}) {
		t.Fatal("CelestiaDAProofValidator address was zero")
	}

	mockCode, err := l1client.CodeAt(ctx, mockBlobstreamAddr, nil)
	Require(t, err)
	if len(mockCode) == 0 {
		t.Fatalf("mock blobstream code missing at %s", mockBlobstreamAddr.Hex())
	}
	validatorCode, err := l1client.CodeAt(ctx, validatorAddr, nil)
	Require(t, err)
	if len(validatorCode) == 0 {
		t.Fatalf("validator code missing at %s", validatorAddr.Hex())
	}

	store := celestiada.NewStore()
	providerServer, providerURL, provider := createCelestiaDAProviderServer(
		t,
		ctx,
		store,
		l1client,
		mockBlobstreamAddr,
		0,
	)
	defer func() { _ = providerServer.Shutdown(context.Background()) }()
	if providerURL == "" {
		t.Fatal("expected Celestia provider URL")
	}
	nodeConfig.DA.ExternalProvider.RPC.URL = providerURL

	certificate, err := provider.Store([]byte("celestia setup verification payload"), 3600).Await(ctx)
	Require(t, err)
	if len(certificate) == 0 {
		t.Fatal("expected non-empty Celestia certificate")
	}
	t.Logf("Generated Celestia certificate (%d bytes)", len(certificate))

	mockTxOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	submitMockBlobstreamForCelestiaCert(t, ctx, l1client, &mockTxOpts, mockBlobstreamAddr, certificate)

	daClient, err := daclient.NewClient(ctx, daclient.TestClientConfig(providerURL), data_streaming.PayloadCommiter())
	Require(t, err)
	dapReaders := daprovider.NewDAProviderRegistry()
	err = dapReaders.SetupDACertificateReader(daClient, daClient)
	Require(t, err)

	_, l2node, l2execNode, _, l2stack, _ := createL2NodeForBoldProtocol(
		t, ctx, true, nodeConfig, l2chainConfig, l2info,
		l1info, l1backend, l1client, l1stack, addresses, stakeTokenAddr,
		false, asserterOpts, signerCfg,
	)
	defer l2node.StopAndWait()
	_, err = execution_consensus.InitAndStartExecutionAndConsensusNodes(ctx, l2stack, l2execNode, l2node)
	Require(t, err)

	go keepChainMoving(t, ctx, l1info, l1client)

	if l2node.InboxTracker == nil {
		t.Fatal("expected inbox tracker")
	}
	batchCount, err := l2node.InboxTracker.GetBatchCount()
	Require(t, err)
	if batchCount == 0 {
		t.Fatal("expected initial inbox batch to be present")
	}

	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
	seqInboxAddr := l1info.GetAddress("SequencerInbox")
	seqInboxBinding, err := bridgegen.NewSequencerInbox(seqInboxAddr, l1client)
	Require(t, err)
	batchData := createBoldBatchData(t, l2info, 5, -1)
	certificate = postBatchWithDA(t, l2node, l1client, &sequencerTxOpts, seqInboxBinding, seqInboxAddr, batchData, provider)
	if len(certificate) == 0 {
		t.Fatal("expected non-empty posted Celestia certificate")
	}
	mockTxOpts = l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	submitMockBlobstreamForCelestiaCert(t, ctx, l1client, &mockTxOpts, mockBlobstreamAddr, certificate)

	newBatchCount, err := l2node.InboxTracker.GetBatchCount()
	Require(t, err)
	if newBatchCount <= batchCount {
		t.Fatalf("expected inbox batch count to increase, before=%d after=%d", batchCount, newBatchCount)
	}

	t.Logf("Celestia setup verified: contracts, provider, writer, validator client, and posted batch all working")
}
