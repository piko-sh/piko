module piko.sh/piko/cmd/piko

go 1.27.0

require (
	charm.land/bubbles/v2 v2.1.1
	charm.land/bubbletea/v2 v2.0.8
	charm.land/lipgloss/v2 v2.0.5
	github.com/NimbleMarkets/ntcharts/v2 v2.2.0
	github.com/charmbracelet/x/ansi v0.11.7
	github.com/google/pprof v0.0.0-20261016150608-1facf8f91b50
	github.com/stretchr/testify v1.12.1
	github.com/valyala/fasthttp v1.73.0
	go.opentelemetry.io/otel v1.47.0
	go.opentelemetry.io/otel/metric v1.47.0
	go.uber.org/goleak v1.3.0
	golang.org/x/mod v0.43.0
	golang.org/x/text v0.44.0
	google.golang.org/grpc v1.84.0
	piko.sh/json v0.1.1
	piko.sh/piko v0.0.0-alpha.47
	piko.sh/piko/wdk/interp/interp_piko_symbols v0.0.0-alpha.47
	piko.sh/piko/wdk/json/json_provider_sonic v0.0.0-alpha.47
	piko.sh/piko/wdk/logger/logger_integration_otel_grpc v0.0.0-alpha.47
	piko.sh/piko/wdk/logger/logger_integration_otel_http v0.0.0-alpha.47
	piko.sh/piko/wdk/logger/logger_otel_sdk v0.0.0-alpha.47
	piko.sh/piko/wdk/monitoring/monitoring_api v0.0.0-alpha.47
	piko.sh/safeconv v0.1.0
	piko.sh/safedisk v0.3.0
	pipit.sh/pipit v0.0.0-alpha.10
	pipit.sh/pipit/sdk/extract v0.0.0-alpha.10
)

require (
	github.com/ThreeDotsLabs/watermill v1.5.2 // indirect
	github.com/andybalholm/brotli v1.3.0 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/bojanz/currency v1.4.4 // indirect
	github.com/bytedance/gopkg v0.1.4 // indirect
	github.com/bytedance/sonic v1.15.3 // indirect
	github.com/bytedance/sonic/loader v0.5.2 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260803092147-8b693049ce2a // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cloudwego/base64x v0.1.7 // indirect
	github.com/cockroachdb/apd/v3 v3.2.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-chi/chi/v5 v5.3.1 // indirect
	github.com/go-chi/cors v1.2.2 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-text/typesetting v0.3.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.31.0 // indirect
	github.com/klauspost/compress v1.20.4 // indirect
	github.com/klauspost/cpuid/v2 v2.4.2 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/lrstanley/bubblezone/v2 v2.0.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-runewidth v0.0.27 // indirect
	github.com/maypok86/otter/v2 v2.3.0 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sony/gobreaker/v2 v2.4.0 // indirect
	github.com/tdewolff/minify/v2 v2.24.15 // indirect
	github.com/tdewolff/parse/v2 v2.8.15 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/quicktemplate v1.8.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.47.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.47.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.1 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/arch v0.29.0 // indirect
	golang.org/x/exp v0.0.0-20261117202309-7a49493bfd90 // indirect
	golang.org/x/image v0.44.0 // indirect
	golang.org/x/net v0.63.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/tools v0.53.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260928230214-8a89bd6388cc // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260928230214-8a89bd6388cc // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	piko.sh/asmgen v0.5.0 // indirect
	piko.sh/goastutil v0.2.0 // indirect
	piko.sh/sharded v0.2.0 // indirect
	piko.sh/stablepool v0.3.0 // indirect
	piko.sh/vectormaths v0.4.0 // indirect
	pipit.sh/pipit/sdk/link v0.0.0-alpha.10 // indirect
)
