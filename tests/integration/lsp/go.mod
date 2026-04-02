module piko.sh/piko/tests/integration/lsp

go 1.26.0

replace piko.sh/piko v0.0.0 => ../../..

replace piko.sh/piko/cmd/lsp v0.0.0 => ../../../cmd/lsp

replace go.lsp.dev/protocol => github.com/politepixels/golang-language-server v0.0.0-20251021203416-834c07056957

require (
	github.com/stretchr/testify v1.11.1
	go.lsp.dev/jsonrpc2 v0.10.0
	go.lsp.dev/protocol v0.12.0
	go.lsp.dev/uri v0.3.0
	go.uber.org/goleak v1.3.0
	piko.sh/piko v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	go.lsp.dev/pkg v0.0.0-20210717090340-384b27a52fb2 // indirect
	golang.org/x/sys v0.42.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
