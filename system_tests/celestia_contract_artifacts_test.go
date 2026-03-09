package arbtest

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tidwall/gjson"
)

func TestCelestiaContractArtifactsEmbedded(t *testing.T) {
	tests := []struct {
		name string
		path string
		data []byte
	}{
		{
			name: "MockBlobstream",
			path: "testdata/celestia/MockBlobstream.json",
			data: mockBlobstreamArtifact,
		},
		{
			name: "CelestiaDAProofValidator",
			path: "testdata/celestia/CelestiaDAProofValidator.json",
			data: celestiaDAProofValidatorArtifact,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.data) == 0 {
				t.Fatalf("%s artifact was not embedded", tt.name)
			}

			fileData, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("read %s: %v", tt.path, err)
			}
			if !bytes.Equal(tt.data, fileData) {
				t.Fatalf("%s embedded bytes do not match %s", tt.name, tt.path)
			}

			abiJSON := gjson.GetBytes(tt.data, "abi").Raw
			if abiJSON == "" {
				t.Fatalf("%s artifact missing abi", tt.name)
			}
			if _, err := abi.JSON(strings.NewReader(abiJSON)); err != nil {
				t.Fatalf("%s abi is invalid: %v", tt.name, err)
			}

			bytecode := gjson.GetBytes(tt.data, "bytecode.object").String()
			if bytecode == "" {
				t.Fatalf("%s artifact missing bytecode.object", tt.name)
			}
			if len(common.FromHex(bytecode)) == 0 {
				t.Fatalf("%s bytecode.object is not valid hex", tt.name)
			}
		})
	}
}
