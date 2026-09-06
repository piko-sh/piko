module piko.sh/piko/tests/integration/lsp

go 1.27.0

replace piko.sh/piko v0.0.0 => ../../..

replace piko.sh/piko/cmd/pikopls v0.0.0 => ../../../cmd/pikopls

require (
	github.com/politepixels/golang-language-server v0.12.1
	github.com/stretchr/testify v1.12.1
	go.lsp.dev/jsonrpc2 v0.10.0
	go.lsp.dev/uri v0.3.0
	go.uber.org/goleak v1.3.0
	piko.sh/piko v0.0.0
)

require (
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	go.lsp.dev/pkg v0.0.0-20210717090340-384b27a52fb2 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.48.0 // indirect
)
