module github.com/offchainlabs/nitro

go 1.26

replace github.com/ethereum/go-ethereum => ./go-ethereum

require (
	cloud.google.com/go/storage v1.56.0
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/Shopify/toxiproxy v2.1.4+incompatible
	github.com/alicebob/miniredis/v2 v2.35.0
	github.com/andybalholm/brotli v1.2.0
	github.com/aws/aws-sdk-go-v2 v1.41.1
	github.com/aws/aws-sdk-go-v2/config v1.32.7
	github.com/aws/aws-sdk-go-v2/credentials v1.19.7
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.21.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.96.0
	github.com/cavaliergopher/grab/v3 v3.0.1
	github.com/ccoveille/go-safecast v1.1.0
	github.com/celestiaorg/celestia-app/v6 v6.4.10
	github.com/celestiaorg/go-square/v3 v3.0.2
	github.com/celestiaorg/nitro-das-celestia v0.7.0-mocha.0.20260312153835-0872b54ed1d9
	github.com/celestiaorg/rsmt2d v0.15.1
	github.com/cockroachdb/pebble v1.1.5
	github.com/codeclysm/extract/v3 v3.0.2
	github.com/cometbft/cometbft v0.38.17
	github.com/enescakir/emoji v1.0.0
	github.com/ethereum/go-ethereum v1.16.8
	github.com/fatih/structtag v1.2.0
	github.com/gdamore/tcell/v2 v2.7.1
	github.com/gobwas/httphead v0.1.0
	github.com/gobwas/ws v1.2.1
	github.com/gobwas/ws-examples v0.0.0-20190625122829-a9e8908d9484
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/google/btree v1.1.3
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/holiman/uint256 v1.3.2
	github.com/jmoiron/sqlx v1.4.0
	github.com/knadh/koanf v1.4.0
	github.com/mailru/easygo v0.0.0-20190618140210-3c14a0dc985f
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/mitchellh/mapstructure v1.5.0
	github.com/pkg/errors v0.9.1
	github.com/r3labs/diff/v3 v3.0.1
	github.com/redis/go-redis/v9 v9.6.3
	github.com/rivo/tview v0.0.0-20240307173318-e804876934a1
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.11.1
	github.com/succinctlabs/sp1-blobstream v1.1.0
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d
	github.com/tidwall/gjson v1.18.0
	github.com/wealdtech/go-merkletree v1.0.0
	go.uber.org/automaxprocs v1.6.0
	golang.org/x/sync v0.19.0
	golang.org/x/sys v0.41.0
	golang.org/x/term v0.40.0
	golang.org/x/tools v0.42.0
	google.golang.org/api v0.266.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require (
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.18.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/iam v1.5.3 // indirect
	cloud.google.com/go/monitoring v1.24.3 // indirect
	cosmossdk.io/api v1.0.0 // indirect
	cosmossdk.io/collections v0.4.0 // indirect
	cosmossdk.io/core v1.1.0 // indirect
	cosmossdk.io/depinject v1.2.1 // indirect
	cosmossdk.io/errors v1.0.2 // indirect
	cosmossdk.io/log v1.6.0 // indirect
	cosmossdk.io/math v1.5.3 // indirect
	cosmossdk.io/store v1.1.2 // indirect
	cosmossdk.io/x/tx v0.13.8 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.30.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.53.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.53.0 // indirect
	github.com/ProjectZKM/Ziren/crates/go-runtime/zkvm_runtime v0.0.0-20251001021608-1fe7b43fc4d6 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.5 // indirect
	github.com/celestiaorg/go-square/v2 v2.3.3 // indirect
	github.com/celestiaorg/merkletree v0.0.0-20230308153949-c33506a7aa26 // indirect
	github.com/celestiaorg/nmt v0.24.2 // indirect
	github.com/cncf/xds/go v0.0.0-20251210132809-ee656c7534f5 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cometbft/cometbft-db v1.0.4 // indirect
	github.com/cosmos/btcutil v1.0.5 // indirect
	github.com/cosmos/cosmos-db v1.1.3 // indirect
	github.com/cosmos/cosmos-proto v1.0.0-beta.5 // indirect
	github.com/cosmos/cosmos-sdk v0.51.6 // indirect
	github.com/cosmos/gogoproto v1.7.2 // indirect
	github.com/cosmos/ics23/go v0.11.0 // indirect
	github.com/crate-crypto/go-eth-kzg v1.4.0 // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/dgraph-io/badger/v4 v4.6.0 // indirect
	github.com/dgraph-io/ristretto/v2 v2.1.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emicklei/dot v1.6.2 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.36.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.0 // indirect
	github.com/ethereum/c-kzg-4844/v2 v2.1.5 // indirect
	github.com/ethereum/go-bigmodexpfix v0.0.0-20250911101455-f9e208c548ab // indirect
	github.com/ethereum/go-verkle v0.2.2 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/ferranbt/fastssz v0.1.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.3 // indirect
	github.com/go-kit/kit v0.13.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/flatbuffers v25.2.10+incompatible // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.11 // indirect
	github.com/googleapis/gax-go/v2 v2.17.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-metrics v0.5.4 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/reedsolomon v1.13.2 // indirect
	github.com/linxGnu/grocksdb v1.9.8 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oasisprotocol/curve25519-voi v0.0.0-20230904125328-1f23a7beb09a // indirect
	github.com/onsi/gomega v1.36.3 // indirect
	github.com/petermattis/goid v0.0.0-20250813065127-a731cc31b4fe // indirect
	github.com/pion/dtls/v2 v2.2.12 // indirect
	github.com/pion/logging v0.2.3 // indirect
	github.com/pion/stun/v2 v2.0.0 // indirect
	github.com/pion/transport/v2 v2.2.10 // indirect
	github.com/pion/transport/v3 v3.0.7 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/sasha-s/go-deadlock v0.3.6 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tendermint/go-amino v0.16.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	go.etcd.io/bbolt v1.4.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.39.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.62.0 // indirect
	go.opentelemetry.io/otel v1.40.0 // indirect
	go.opentelemetry.io/otel/metric v1.40.0 // indirect
	go.opentelemetry.io/otel/sdk v1.40.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.40.0 // indirect
	go.opentelemetry.io/otel/trace v1.40.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/exp v0.0.0-20260112195511-716be5621a96 // indirect
	golang.org/x/text v0.34.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	google.golang.org/genproto v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260203192932-546029d2fa20 // indirect
	google.golang.org/grpc v1.79.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

require (
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/VictoriaMetrics/fastcache v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.6 // indirect
	github.com/aws/smithy-go v1.24.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.24.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/errors v1.12.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.6 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/consensys/gnark-crypto v0.19.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240724233137-53bbb0ceb27a // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/deckarep/golang-set/v2 v2.6.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/dop251/goja v0.0.0-20230806174421-c933cf95e127 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gammazero/deque v1.1.0 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/getsentry/sentry-go v0.42.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/gogo/protobuf v1.3.3 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-github/v62 v62.0.0
	github.com/google/pprof v0.0.0-20250820193118-f64d9cf942d6 // indirect
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/graph-gophers/graphql-go v1.3.0 // indirect
	github.com/h2non/filetype v1.0.6 // indirect
	github.com/hashicorp/go-bexpr v0.1.10 // indirect
	github.com/holiman/billy v0.0.0-20250707135307-f2f9b9aae7db // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/huin/goupnp v1.3.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/juju/errors v0.0.0-20181118221551-089d3ea4e4d5 // indirect
	github.com/juju/loggo v0.0.0-20180524022052-584905176618 // indirect
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/pointerstructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/rhnvrm/simples3 v0.6.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/supranational/blst v0.3.16-0.20250831170142-f48500c1fdbe // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/urfave/cli/v2 v2.27.5 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/mod v0.33.0
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/oauth2 v0.35.0
)

replace (
	cosmossdk.io/x/upgrade => github.com/celestiaorg/cosmos-sdk/x/upgrade v0.2.0
	github.com/cometbft/cometbft => github.com/celestiaorg/celestia-core v0.39.24
	github.com/cosmos/cosmos-sdk => github.com/celestiaorg/cosmos-sdk v0.52.0
	github.com/cosmos/ibc-go/v8 => github.com/celestiaorg/ibc-go/v8 v8.7.2
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

	// Local bridgegen replacement
	github.com/offchainlabs/nitro/solgen/go/bridgegen => ./daserver/bridgegen
	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	// celestia-core(v0.34.x): used for multiplexing abci v1 requests
	github.com/tendermint/tendermint => github.com/celestiaorg/celestia-core v1.55.0-tm-v0.34.35
)

replace github.com/ipfs/boxo => github.com/celestiaorg/boxo v0.29.0-fork-4

// Force the version that go-ethereum 1.15.6 actually needs
replace github.com/crate-crypto/go-kzg-4844 => github.com/crate-crypto/go-kzg-4844 v1.1.0

replace github.com/ipfs/go-datastore => github.com/celestiaorg/go-datastore v0.0.0-20250801131506-48a63ae531e4

replace cosmossdk.io/api => github.com/celestiaorg/cosmos-sdk/api v0.7.6

replace cosmossdk.io/log => github.com/celestiaorg/cosmos-sdk/log v1.1.1-0.20251116153902-f48fea92e627
