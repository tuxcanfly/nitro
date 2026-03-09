package celestiamock

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type jsonrpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func rawCall(t *testing.T, url string, body string) jsonrpcResp {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http call failed: %v", err)
	}
	defer resp.Body.Close()
	var out jsonrpcResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func TestJSONRPCHexUint64Compatibility(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, ln, err := StartRPC(ctx, "127.0.0.1", 0, NewServer(NewStore()))
	if err != nil {
		t.Fatalf("start rpc: %v", err)
	}
	defer ln.Close()
	url := "http://" + ln.Addr().String()

	// Same shape expected by Nitro DA client: timeout as hex string.
	resp := rawCall(t, url, `{"jsonrpc":"2.0","id":1,"method":"daprovider_store","params":["0x112233","0x3c"]}`)
	if resp.Error != nil {
		t.Fatalf("unexpected rpc error: %+v", resp.Error)
	}
	var storeResult struct {
		SerializedDACert hexutil.Bytes `json:"serialized-da-cert"`
	}
	if err := json.Unmarshal(resp.Result, &storeResult); err != nil {
		t.Fatalf("unmarshal store result: %v", err)
	}
	if len(storeResult.SerializedDACert) == 0 {
		t.Fatalf("empty cert")
	}

	seqMsg := "0x01" + strings.TrimPrefix(storeResult.SerializedDACert.String(), "0x")
	resp2 := rawCall(t, url, `{"jsonrpc":"2.0","id":2,"method":"daprovider_recoverPayload","params":["0x1","0x0000000000000000000000000000000000000000000000000000000000000000","`+seqMsg+`"]}`)
	if resp2.Error != nil {
		t.Fatalf("unexpected recover error: %+v", resp2.Error)
	}
	var payloadResult struct {
		Payload []byte `json:"payload"`
	}
	if err := json.Unmarshal(resp2.Result, &payloadResult); err != nil {
		t.Fatalf("unmarshal payload result: %v", err)
	}
	if string(payloadResult.Payload) != string([]byte{0x11, 0x22, 0x33}) {
		t.Fatalf("payload mismatch: %x", payloadResult.Payload)
	}

	// Also ensure typed rpc client can call it.
	client, err := rpc.DialHTTP(url)
	if err != nil {
		t.Fatalf("dial rpc: %v", err)
	}
	defer client.Close()
	var maxRes struct {
		MaxSize int `json:"maxSize"`
	}
	if err := client.Call(&maxRes, "daprovider_getMaxMessageSize"); err != nil {
		t.Fatalf("typed call getMaxMessageSize: %v", err)
	}
	if maxRes.MaxSize <= 0 {
		t.Fatalf("bad max size: %d", maxRes.MaxSize)
	}

	time.Sleep(5 * time.Millisecond)
}
