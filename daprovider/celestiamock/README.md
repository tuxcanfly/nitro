# mockcelestiadas

Lightweight mock for a Celestia-style DA RPC endpoint used by Nitro.

This package is intentionally simple:
- No auth token checks
- In-memory storage
- Implements `daprovider_*` RPC methods expected by Nitro DA client
- Usable as both a reusable Go library and standalone executable

## Library

Package: `github.com/offchainlabs/nitro/daprovider/celestiamock`

Main entry points:
- `celestiamock.NewStore()`
- `celestiamock.NewServer(store)`
- `celestiamock.StartRPC(ctx, addr, port, srv)`

## Executable

Binary entrypoint:
- `./cmd/mockcelestiadas`

Run:

```bash
cd nitro
go run ./cmd/mockcelestiadas --addr 0.0.0.0 --port 9880
```

Build:

```bash
cd nitro
go build -o target/bin/mockcelestiadas ./cmd/mockcelestiadas
```

## Nitro config example

Point external DA provider RPC to this mock:

```json
{
  "node": {
    "da": {
      "external-provider": {
        "enable": true,
        "with-writer": true,
        "rpc": {
          "url": "http://127.0.0.1:9880"
        }
      }
    }
  }
}
```

## Implemented RPC methods

- `daprovider_store`
- `daprovider_getSupportedHeaderBytes`
- `daprovider_getMaxMessageSize`
- `daprovider_recoverPayload`
- `daprovider_collectPreimages`
- `daprovider_recoverPayloadAndPreimages`
- `daprovider_generateCertificateValidityProof`
- `daprovider_generateReadPreimageProof`

Note: proof formats are mock/dev oriented and not equivalent to production Celestia proof verification.
