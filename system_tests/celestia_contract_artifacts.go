package arbtest

import _ "embed"

var (
	//go:embed testdata/celestia/MockBlobstream.json
	mockBlobstreamArtifact []byte

	//go:embed testdata/celestia/CelestiaDAProofValidator.json
	celestiaDAProofValidatorArtifact []byte
)
