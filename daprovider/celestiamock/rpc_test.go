package celestiamock

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestServerEndToEndJSONRPC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := NewServer(NewStore())
	_, ln, err := StartRPC(ctx, "127.0.0.1", 0, srv)
	if err != nil {
		t.Fatalf("StartRPC failed: %v", err)
	}
	defer ln.Close()

	client, err := rpc.DialHTTP("http://" + ln.Addr().String())
	if err != nil {
		t.Fatalf("DialHTTP failed: %v", err)
	}
	defer client.Close()

	msg := []byte("hello-mock-celestia")
	var storeRes struct {
		SerializedDACert hexutil.Bytes `json:"serialized-da-cert"`
	}
	if err := client.Call(&storeRes, "daprovider_store", hexutil.Bytes(msg), hexutil.Uint64(60)); err != nil {
		t.Fatalf("daprovider_store failed: %v", err)
	}
	if len(storeRes.SerializedDACert) == 0 {
		t.Fatalf("empty serialized-da-cert")
	}

	seqMsg := append([]byte{DACertificateMessageHeaderFlag}, storeRes.SerializedDACert...)
	var recRes struct {
		Payload []byte `json:"payload"`
	}
	if err := client.Call(&recRes, "daprovider_recoverPayload", hexutil.Uint64(1), common.Hash{}, hexutil.Bytes(seqMsg)); err != nil {
		t.Fatalf("daprovider_recoverPayload failed: %v", err)
	}
	if string(recRes.Payload) != string(msg) {
		t.Fatalf("payload mismatch got=%q want=%q", string(recRes.Payload), string(msg))
	}

	var validityRes struct {
		Proof hexutil.Bytes `json:"proof"`
	}
	if err := client.Call(&validityRes, "daprovider_generateCertificateValidityProof", storeRes.SerializedDACert); err != nil {
		t.Fatalf("daprovider_generateCertificateValidityProof failed: %v", err)
	}
	if len(validityRes.Proof) != 2 || validityRes.Proof[0] != 0x01 {
		t.Fatalf("unexpected validity proof: %x", []byte(validityRes.Proof))
	}

	var readProofRes struct {
		Proof hexutil.Bytes `json:"proof"`
	}
	if err := client.Call(&readProofRes, "daprovider_generateReadPreimageProof", hexutil.Uint64(0), storeRes.SerializedDACert); err != nil {
		t.Fatalf("daprovider_generateReadPreimageProof failed: %v", err)
	}
	if len(readProofRes.Proof) < 10 {
		t.Fatalf("read proof too short: %d", len(readProofRes.Proof))
	}

	time.Sleep(10 * time.Millisecond)
}
