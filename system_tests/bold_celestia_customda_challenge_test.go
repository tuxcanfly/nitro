// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	celestiacert "github.com/celestiaorg/nitro-das-celestia/daserver/cert"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/bold/challenge"
	modes "github.com/offchainlabs/nitro/bold/challenge/types"
	"github.com/offchainlabs/nitro/bold/protocol/sol"
	"github.com/offchainlabs/nitro/bold/state"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/celestiada"
	"github.com/offchainlabs/nitro/daprovider/daclient"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/execution_consensus"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/validator/proofenhancement"
	"github.com/offchainlabs/nitro/validator/valnode"
)

type CelestiaCustomDAEvilStrategy int

const (
	CelestiaEvilDataGoodCert CelestiaCustomDAEvilStrategy = iota
	CelestiaEvilDataEvilCert
	CelestiaInvalidCertClaimedValid
	CelestiaValidCertClaimedInvalid
)

func TestChallengeProtocolBOLDCelestiaDA_EvilDataGoodCert(t *testing.T) {
	testChallengeProtocolBOLDCelestiaDA(t, CelestiaEvilDataGoodCert)
}

func TestChallengeProtocolBOLDCelestiaDA_EvilDataEvilCert(t *testing.T) {
	testChallengeProtocolBOLDCelestiaDA(t, CelestiaEvilDataEvilCert)
}

// Celestia certificates are not signer-based; this covers the equivalent
// "invalid cert claimed valid" divergence used by the validity OSP path.
func TestChallengeProtocolBOLDCelestiaDA_InvalidCertClaimedValid(t *testing.T) {
	testChallengeProtocolBOLDCelestiaDA(t, CelestiaInvalidCertClaimedValid)
}

func TestChallengeProtocolBOLDCelestiaDA_ValidCertClaimedInvalid(t *testing.T) {
	testChallengeProtocolBOLDCelestiaDA(t, CelestiaValidCertClaimedInvalid)
}

func testChallengeProtocolBOLDCelestiaDA(t *testing.T, evilStrategy CelestiaCustomDAEvilStrategy) {
	goodDir, err := os.MkdirTemp("", "celestia_good_*")
	Require(t, err)
	evilDir, err := os.MkdirTemp("", "celestia_evil_*")
	Require(t, err)
	t.Cleanup(func() {
		Require(t, os.RemoveAll(goodDir))
		Require(t, os.RemoveAll(evilDir))
	})

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

	nodeConfigA := arbnode.ConfigDefaultL1Test()
	nodeConfigA.DA.ExternalProvider.Enable = true

	l1info, l1backend, l1client, l1stack, addresses, stakeTokenAddr, asserterOpts, signerCfg := setupL1ForBoldProtocol(
		t, ctx, sconf, l2info, false, nodeConfigA, l2chainConfig, true, true,
	)
	defer requireClose(t, l1stack)

	mockBlobstreamAddr := l1info.GetAddress("MockBlobstream")
	validatorAddr := l1info.GetAddress("CelestiaDAProofValidator")
	if mockBlobstreamAddr == (common.Address{}) || validatorAddr == (common.Address{}) {
		t.Fatal("Celestia validator contracts were not deployed")
	}

	store := celestiada.NewStore()
	providerServerA, providerURLNodeA, daWriter := createCelestiaDAProviderServer(
		t,
		ctx,
		store,
		l1client,
		mockBlobstreamAddr,
		0,
	)
	defer func() { _ = providerServerA.Shutdown(context.Background()) }()

	providerServerB, providerURLNodeB, evilProvider := createEvilCelestiaDAProviderServer(
		t,
		ctx,
		store,
		l1client,
		mockBlobstreamAddr,
		0,
	)
	defer func() { _ = providerServerB.Shutdown(context.Background()) }()

	nodeConfigA.DA.ExternalProvider.RPC.URL = providerURLNodeA

	l2info, l2nodeA, l2execNodeA, _, l2stackA, assertionChain := createL2NodeForBoldProtocol(
		t, ctx, true, nodeConfigA, l2chainConfig, l2info,
		l1info, l1backend, l1client, l1stack, addresses, stakeTokenAddr,
		false, asserterOpts, signerCfg,
	)
	defer l2nodeA.StopAndWait()

	go keepChainMoving(t, ctx, l1info, l1client)

	l2nodeConfig := arbnode.ConfigDefaultL1Test()
	l2nodeConfig.DA.ExternalProvider.Enable = true
	l2nodeConfig.DA.ExternalProvider.RPC.URL = providerURLNodeB

	l2clientB, l2nodeB, l2execNodeB, l2stackB := createNodeBWithSharedContracts(
		t,
		ctx,
		l2nodeA,
		l1stack,
		l1info,
		&l2info.ArbInitData,
		l2nodeConfig,
		nil,
		sconf,
		stakeTokenAddr,
		l1client,
		assertionChain,
	)
	defer l2nodeB.StopAndWait()
	_ = l2clientB

	genesisA, err := l2nodeA.ExecutionClient.ResultAtMessageIndex(0).Await(ctx)
	Require(t, err)
	genesisB, err := l2nodeB.ExecutionClient.ResultAtMessageIndex(0).Await(ctx)
	Require(t, err)
	if genesisA.BlockHash != genesisB.BlockHash {
		Fatal(t, "genesis blocks mismatch between nodes")
	}

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	TransferBalance(t, "Faucet", "Asserter", balance, l1info, l1client, ctx)
	TransferBalance(t, "Faucet", "EvilAsserter", balance, l1info, l1client, ctx)

	valCfg := valnode.TestValidationConfig
	valCfg.UseJit = false
	blockValidatorConfig := staker.TestBlockValidatorConfig
	currentWasmModuleRoot := currentRootModule(t)
	_, valStackA := createTestValidationNode(t, ctx, &valCfg)
	_, valStackB := createTestValidationNode(t, ctx, &valCfg)

	daClientA, err := daclient.NewClient(ctx, daclient.TestClientConfig(providerURLNodeA), data_streaming.PayloadCommiter())
	Require(t, err)
	daClientB, err := daclient.NewClient(ctx, daclient.TestClientConfig(providerURLNodeB), data_streaming.PayloadCommiter())
	Require(t, err)

	dapReadersA := daprovider.NewDAProviderRegistry()
	Require(t, dapReadersA.SetupDACertificateReader(daClientA, daClientA))
	dapReadersB := daprovider.NewDAProviderRegistry()
	Require(t, dapReadersB.SetupDACertificateReader(daClientB, daClientB))

	statelessA, err := staker.NewStatelessBlockValidator(
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		l2nodeA.ExecutionRecorder,
		l2nodeA.ConsensusDB,
		dapReadersA,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStackA,
		currentWasmModuleRoot,
	)
	Require(t, err)
	Require(t, statelessA.Start(ctx))

	statelessB, err := staker.NewStatelessBlockValidator(
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.ExecutionRecorder,
		l2nodeB.ConsensusDB,
		dapReadersB,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStackB,
		currentWasmModuleRoot,
	)
	Require(t, err)
	Require(t, statelessB.Start(ctx))

	blockValidatorA, err := staker.NewBlockValidator(statelessA, l2nodeA.InboxTracker, l2nodeA.TxStreamer, StaticFetcherFrom(t, &blockValidatorConfig), nil)
	Require(t, err)
	Require(t, blockValidatorA.Initialize(ctx))
	Require(t, blockValidatorA.Start(ctx))

	blockValidatorB, err := staker.NewBlockValidator(statelessB, l2nodeB.InboxTracker, l2nodeB.TxStreamer, StaticFetcherFrom(t, &blockValidatorConfig), nil)
	Require(t, err)
	Require(t, blockValidatorB.Initialize(ctx))
	Require(t, blockValidatorB.Start(ctx))

	proofEnhancerA := proofenhancement.NewProofEnhancementManager()
	customDAEnhancerA := proofenhancement.NewReadPreimageProofEnhancer(dapReadersA, l2nodeA.InboxTracker, l2nodeA.InboxReader)
	proofEnhancerA.RegisterEnhancer(proofenhancement.MarkerCustomDAReadPreimage, customDAEnhancerA)
	validateCertificateEnhancerA := proofenhancement.NewValidateCertificateProofEnhancer(dapReadersA, l2nodeA.InboxTracker, l2nodeA.InboxReader)
	proofEnhancerA.RegisterEnhancer(proofenhancement.MarkerCustomDAValidateCertificate, validateCertificateEnhancerA)

	proofEnhancerB := proofenhancement.NewProofEnhancementManager()
	customDAEnhancerB := proofenhancement.NewReadPreimageProofEnhancer(dapReadersB, l2nodeB.InboxTracker, l2nodeB.InboxReader)
	validateCertificateEnhancerB := proofenhancement.NewValidateCertificateProofEnhancer(dapReadersB, l2nodeB.InboxTracker, l2nodeB.InboxReader)
	proofEnhancerB.RegisterEnhancer(proofenhancement.MarkerCustomDAValidateCertificate, validateCertificateEnhancerB)

	var evilEnhancer *EvilCustomDAProofEnhancer
	if evilStrategy == CelestiaEvilDataEvilCert {
		evilEnhancer = NewEvilCustomDAProofEnhancer(customDAEnhancerB)
		proofEnhancerB.RegisterEnhancer(proofenhancement.MarkerCustomDAReadPreimage, evilEnhancer)
	} else {
		proofEnhancerB.RegisterEnhancer(proofenhancement.MarkerCustomDAReadPreimage, customDAEnhancerB)
	}

	stateManagerA, err := bold.NewBOLDStateProvider(
		blockValidatorA,
		statelessA,
		state.Height(blockChallengeLeafHeight),
		&bold.StateProviderConfig{ValidatorName: "good-celestia", MachineLeavesCachePath: goodDir, CheckBatchFinality: false},
		goodDir,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		l2nodeA.InboxReader,
		proofEnhancerA,
	)
	Require(t, err)

	stateManagerB, err := bold.NewBOLDStateProvider(
		blockValidatorB,
		statelessB,
		state.Height(blockChallengeLeafHeight),
		&bold.StateProviderConfig{ValidatorName: "evil-celestia", MachineLeavesCachePath: evilDir, CheckBatchFinality: false},
		evilDir,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		l2nodeB.InboxReader,
		proofEnhancerB,
	)
	Require(t, err)

	_, err = execution_consensus.InitAndStartExecutionAndConsensusNodes(ctx, l2stackA, l2execNodeA, l2nodeA)
	Require(t, err)
	_, err = execution_consensus.InitAndStartExecutionAndConsensusNodes(ctx, l2stackB, l2execNodeB, l2nodeB)
	Require(t, err)

	chalManagerAddr := assertionChain.SpecChallengeManager()
	evilOpts := l1info.GetDefaultTransactOpts("EvilAsserter", ctx)
	l1ChainID, err := l1client.ChainID(ctx)
	Require(t, err)
	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2nodeB.ConsensusDB, storage.StakerPrefix),
		l2nodeB.L1Reader,
		&evilOpts,
		NewCommonConfigFetcher(l2nodeConfig),
		l2nodeB.SyncMonitor,
		l1ChainID,
	)
	Require(t, err)
	chainB, err := sol.NewAssertionChain(
		ctx,
		assertionChain.RollupAddress(),
		chalManagerAddr.Address(),
		&evilOpts,
		l1client,
		bold.NewDataPosterTransactor(dp),
		sol.WithRpcHeadBlockNumber(rpc.LatestBlockNumber),
	)
	Require(t, err)

	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
	seqInboxAddr := l1info.GetAddress("SequencerInbox")
	seqInboxBinding, err := bridgegen.NewSequencerInbox(seqInboxAddr, l1client)
	Require(t, err)

	totalMessagesPosted := int64(0)
	firstBatchData := createBoldBatchData(t, l2info, 5, -1)
	firstCert := postBatchWithDA(t, l2nodeA, l1client, &sequencerTxOpts, seqInboxBinding, seqInboxAddr, firstBatchData, daWriter)
	mockTxOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	submitMockBlobstreamForCelestiaCert(t, ctx, l1client, &mockTxOpts, mockBlobstreamAddr, firstCert)
	totalMessagesPosted += 5

	l2info.Accounts["Owner"].Nonce.Store(5)
	goodBatchData2 := createBoldBatchData(t, l2info, 10, -1)
	l2info.Accounts["Owner"].Nonce.Store(5)
	evilBatchData2 := createBoldBatchData(t, l2info, 10, 5)

	goodCert2, err := daWriter.Store(goodBatchData2, 3600).Await(ctx)
	Require(t, err)
	mockTxOpts = l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	submitMockBlobstreamForCelestiaCert(t, ctx, l1client, &mockTxOpts, mockBlobstreamAddr, goodCert2)

	certToPost := goodCert2
	goodCertKeccak := celestiaCertHash(goodCert2)

	switch evilStrategy {
	case CelestiaEvilDataGoodCert, CelestiaEvilDataEvilCert:
		evilCert, err := daWriter.Store(evilBatchData2, 3600).Await(ctx)
		Require(t, err)
		mockTxOpts = l1info.GetDefaultTransactOpts("RollupOwner", ctx)
		submitMockBlobstreamForCelestiaCert(t, ctx, l1client, &mockTxOpts, mockBlobstreamAddr, evilCert)
		evilProvider.SetReadAlias(goodCertKeccak, evilCert)
		evilProvider.SetEvilPayload(goodCertKeccak, evilBatchData2)
		evilProvider.SetReadPreimageProofAlias(goodCertKeccak, evilCert)
		if evilStrategy == CelestiaEvilDataEvilCert {
			if evilEnhancer != nil {
				evilEnhancer.SetMapping(goodCertKeccak, evilCert)
			}
		}
	case CelestiaInvalidCertClaimedValid:
		invalidCert := mutateCelestiaCertDataRoot(goodCert2)
		invalidCertKeccak := celestiaCertHash(invalidCert)
		evilProvider.SetReadAlias(invalidCertKeccak, goodCert2)
		evilProvider.SetReadPreimageProofAlias(invalidCertKeccak, goodCert2)
		evilProvider.SetClaimCertValid(invalidCertKeccak)
		certToPost = invalidCert
	case CelestiaValidCertClaimedInvalid:
		evilProvider.SetClaimCertInvalid(goodCertKeccak)
	}

	if evilStrategy == CelestiaEvilDataGoodCert || evilStrategy == CelestiaEvilDataEvilCert {
		sequencerMsg := make([]byte, celestiacert.SequencerMsgOffset+len(goodCert2))
		copy(sequencerMsg[celestiacert.SequencerMsgOffset:], goodCert2)

		honestPayload, err := daClientA.RecoverPayload(0, common.Hash{}, sequencerMsg).Await(ctx)
		Require(t, err)
		if !bytes.Equal(honestPayload.Payload, goodBatchData2) {
			t.Fatalf("honest Celestia DA provider returned unexpected payload for good cert")
		}

		evilPayload, err := daClientB.RecoverPayload(0, common.Hash{}, sequencerMsg).Await(ctx)
		Require(t, err)
		if !bytes.Equal(evilPayload.Payload, evilBatchData2) {
			t.Fatalf("evil Celestia DA provider did not return evil payload for good cert")
		}
	}

	receipt := postBatchToL1(t, ctx, l1client, &sequencerTxOpts, seqInboxBinding, certToPost)
	syncBatchToNode(t, ctx, l1client, l2nodeA, seqInboxAddr, receipt, "")
	syncBatchToNode(t, ctx, l1client, l2nodeB, seqInboxAddr, receipt, "")
	totalMessagesPosted += 10

	bcA, err := l2nodeA.InboxTracker.GetBatchCount()
	Require(t, err)
	bcB, err := l2nodeB.InboxTracker.GetBatchCount()
	Require(t, err)
	if bcA != bcB {
		t.Fatalf("expected equal batch counts, got A=%d B=%d", bcA, bcB)
	}

	msgA, err := l2nodeA.InboxTracker.GetBatchMessageCount(bcA - 1)
	Require(t, err)
	msgB, err := l2nodeB.InboxTracker.GetBatchMessageCount(bcB - 1)
	Require(t, err)

	rejectedByNodeA := evilStrategy == CelestiaInvalidCertClaimedValid
	rejectedByNodeB := evilStrategy == CelestiaValidCertClaimedInvalid

	switch {
	case rejectedByNodeA:
		if msgA != (msgB-10)+1 {
			t.Fatalf("expected node A to reject divergent cert batch, got A=%d B=%d", msgA, msgB)
		}
	case rejectedByNodeB:
		if msgB != (msgA-10)+1 {
			t.Fatalf("expected node B to reject divergent cert batch, got A=%d B=%d", msgA, msgB)
		}
	default:
		if msgA != msgB {
			t.Fatalf("expected equal message counts, got A=%d B=%d", msgA, msgB)
		}
	}

	deadline := time.After(90 * time.Second)
	nodeAExpected := uint64(totalMessagesPosted)
	nodeBExpected := uint64(totalMessagesPosted)
	switch {
	case rejectedByNodeA:
		nodeAExpected = uint64(totalMessagesPosted-10) + 1
	case rejectedByNodeB:
		nodeBExpected = uint64(totalMessagesPosted-10) + 1
	}
	for {
		select {
		case <-deadline:
			resultA, errA := l2nodeA.ExecutionClient.ResultAtMessageIndex(arbutil.MessageIndex(nodeAExpected)).Await(ctx)
			resultB, errB := l2nodeB.ExecutionClient.ResultAtMessageIndex(arbutil.MessageIndex(nodeBExpected)).Await(ctx)
			t.Fatalf(
				"timed out waiting for Celestia nodes to diverge: msgResultA(idx=%d)=%v errA=%v msgResultB(idx=%d)=%v errB=%v",
				nodeAExpected, resultA, errA,
				nodeBExpected, resultB, errB,
			)
		default:
		}

		resultA, errA := l2nodeA.ExecutionClient.ResultAtMessageIndex(arbutil.MessageIndex(nodeAExpected)).Await(ctx)
		resultB, errB := l2nodeB.ExecutionClient.ResultAtMessageIndex(arbutil.MessageIndex(nodeBExpected)).Await(ctx)
		if errA != nil || errB != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if resultA.BlockHash == resultB.BlockHash {
			Fatal(t, "Celestia nodes unexpectedly converged after adversarial batch")
		}
		break
	}

	bridgeBinding, err := bridgegen.NewBridge(l1info.GetAddress("Bridge"), l1client)
	Require(t, err)
	totalBatchesBig, err := bridgeBinding.SequencerMessageCount(&bind.CallOpts{Context: ctx})
	Require(t, err)
	totalBatches := totalBatchesBig.Uint64()

	validatorATarget := totalBatches - 1
	if rejectedByNodeA {
		validatorATarget = totalBatches - 2
	}
	validatorBTarget := totalBatches - 1
	if rejectedByNodeB {
		validatorBTarget = totalBatches - 2
	}

	waitForValidatedBatch(t, blockValidatorA, validatorATarget)
	waitForValidatedBatch(t, blockValidatorB, validatorBTarget)

	provider := state.NewHistoryCommitmentProvider(
		stateManagerA,
		stateManagerA,
		stateManagerA,
		[]state.Height{
			state.Height(blockChallengeLeafHeight),
			state.Height(bigStepChallengeLeafHeight),
			state.Height(bigStepChallengeLeafHeight),
			state.Height(bigStepChallengeLeafHeight),
			state.Height(smallStepChallengeLeafHeight),
		},
		stateManagerA,
		nil,
	)
	evilHistoryProvider := state.NewHistoryCommitmentProvider(
		stateManagerB,
		stateManagerB,
		stateManagerB,
		[]state.Height{
			state.Height(blockChallengeLeafHeight),
			state.Height(bigStepChallengeLeafHeight),
			state.Height(bigStepChallengeLeafHeight),
			state.Height(bigStepChallengeLeafHeight),
			state.Height(smallStepChallengeLeafHeight),
		},
		stateManagerB,
		nil,
	)

	stackOpts := []challenge.StackOpt{
		challenge.StackWithName("honest-celestia-customda"),
		challenge.StackWithMode(modes.MakeMode),
		challenge.StackWithPostingInterval(3 * time.Second),
		challenge.StackWithPollingInterval(time.Second),
		challenge.StackWithMinimumGapToParentAssertion(0),
		challenge.StackWithAverageBlockCreationTime(time.Second),
	}
	managerA, err := challenge.NewChallengeStack(assertionChain, provider, stackOpts...)
	Require(t, err)
	managerB, err := challenge.NewChallengeStack(chainB, evilHistoryProvider, append(stackOpts, challenge.StackWithName("evil-celestia-customda"))...)
	Require(t, err)
	managerA.Start(ctx)
	managerB.Start(ctx)

	chalManager := assertionChain.SpecChallengeManager()
	filterer, err := challengeV2gen.NewEdgeChallengeManagerFilterer(chalManager.Address(), l1client)
	Require(t, err)

	fromBlock := uint64(0)
	expectedOSPWinner := l1info.GetDefaultTransactOpts("Asserter", ctx).From
	expectedOSPWinnerLabel := "honest"
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestBlock, err := l1client.HeaderByNumber(ctx, nil)
			Require(t, err)
			toBlock := latestBlock.Number.Uint64()
			if fromBlock == toBlock {
				continue
			}
			it, err := filterer.FilterEdgeConfirmedByOneStepProof(&bind.FilterOpts{Start: fromBlock, End: &toBlock, Context: ctx}, nil, nil)
			Require(t, err)
			for it.Next() {
				if it.Error() != nil {
					t.Fatalf("iterator error: %v", it.Error())
				}
				t.Log("Received event of OSP confirmation!")
				tx, _, err := l1client.TransactionByHash(ctx, it.Event.Raw.TxHash)
				Require(t, err)
				signer := types.NewCancunSigner(tx.ChainId())
				winner, err := signer.Sender(tx)
				Require(t, err)
				if winner == expectedOSPWinner {
					t.Logf("%s party won OSP", expectedOSPWinnerLabel)
					Require(t, it.Close())
					time.Sleep(5 * time.Second)
					return
				}
				if winner == l1info.GetDefaultTransactOpts("Asserter", ctx).From || winner == l1info.GetDefaultTransactOpts("EvilAsserter", ctx).From {
					t.Fatalf("unexpected OSP winner %s for strategy %v", winner.Hex(), evilStrategy)
				}
			}
			fromBlock = toBlock
		case <-ctx.Done():
			t.Fatal("context cancelled before Celestia OSP confirmation")
		}
	}
}

func waitForValidatedBatch(t *testing.T, validator *staker.BlockValidator, target uint64) {
	t.Helper()
	deadline := time.After(90 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for validator to reach batch %d", target)
		default:
		}
		lastInfo, err := validator.ReadLastValidatedInfo()
		if err == nil && lastInfo != nil && lastInfo.GlobalState.Batch >= target {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}
