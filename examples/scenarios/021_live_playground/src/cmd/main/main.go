package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "testmodule/dist"

	"piko.sh/piko"
	"piko.sh/piko/wdk/logger"
)

func main() {
	logger.AddPrettyOutput()

	// Serve WASM artefacts from bin/wasm/ (built by `make build-wasm`) on a
	// separate port. The playground page fetches these from this server.
	// Serves pre-compressed brotli or gzip variants when the client supports
	// them, falling back to the uncompressed binary.
	wasmDir := findWASMDir()
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/piko.wasm", serveWASM(wasmDir))
		mux.HandleFunc("/wasm_exec.js", servePrecompressed(wasmDir, "wasm_exec.js", "text/javascript"))
		fmt.Println("WASM static server at http://localhost:3001 (serving from " + wasmDir + ")")
		if err := http.ListenAndServe(":3001", mux); err != nil {
			fmt.Fprintf(os.Stderr, "static server error: %v\n", err)
		}
	}()

	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	ssr := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
		piko.WithDevWidget(),
		piko.WithDevHotreload(),
		piko.WithMonitoring(),
		piko.WithCSP(func(b *piko.CSPBuilder) {
			b.DefaultSrc(piko.CSPSelf).
				ScriptSrc(
					piko.CSPSelf,
					piko.CSPUnsafeEval,
					piko.CSPHost("https://cdn.jsdelivr.net"),
					piko.CSPHost("https://esm.sh"),
					piko.CSPHost("http://localhost:3001"),
				).
				ScriptSrcElem(
					piko.CSPSelf,
					piko.CSPUnsafeInline,
					piko.CSPHost("https://cdn.jsdelivr.net"),
					piko.CSPHost("https://esm.sh"),
					piko.CSPHost("http://localhost:3001"),
				).
				StyleSrc(piko.CSPSelf, piko.CSPUnsafeInline, piko.CSPHost("https://cdn.jsdelivr.net")).
				FontSrc(piko.CSPSelf, piko.CSPData, piko.CSPHost("https://cdn.jsdelivr.net")).
				WorkerSrc(piko.CSPSelf, piko.CSPBlob).
				ConnectSrc(piko.CSPSelf, piko.CSPHost("http://localhost:3001"), piko.CSPHost("https://cdn.jsdelivr.net"), piko.CSPHost("https://esm.sh")).
				ImgSrc(piko.CSPSelf, piko.CSPData, piko.CSPBlob)
		}),
	)
	if err := ssr.Run(command); err != nil {
		panic(err)
	}
}

// serveWASM returns a handler that serves the WASM binary with content
// negotiation for pre-compressed variants. Prefers brotli, then gzip,
// falling back to the uncompressed binary.
func serveWASM(wasmDir string) http.HandlerFunc {
	return servePrecompressed(wasmDir, "piko.final.wasm", "application/wasm")
}

// servePrecompressed returns a handler that serves a file from wasmDir,
// preferring pre-compressed .br or .gz variants based on Accept-Encoding.
func servePrecompressed(wasmDir, filename, contentType string) http.HandlerFunc {
	rawPath := filepath.Join(wasmDir, filename)
	brPath := rawPath + ".br"
	gzPath := rawPath + ".gz"

	hasBrotli := fileExists(brPath)
	hasGzip := fileExists(gzPath)

	return func(w http.ResponseWriter, r *http.Request) {
		setCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		accept := r.Header.Get("Accept-Encoding")
		w.Header().Set("Content-Type", contentType)

		if hasBrotli && strings.Contains(accept, "br") {
			w.Header().Set("Content-Encoding", "br")
			http.ServeFile(w, r, brPath)
			return
		}
		if hasGzip && strings.Contains(accept, "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			http.ServeFile(w, r, gzPath)
			return
		}
		http.ServeFile(w, r, rawPath)
	}
}

// setCORS adds permissive CORS headers for the local dev server.
func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
}

// fileExists reports whether path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// findWASMDir locates the bin/wasm/ directory by walking up from the
// current working directory to find the repository root.
func findWASMDir() string {
	dir, err := os.Getwd()
	if err != nil {
		panic("cannot determine working directory: " + err.Error())
	}
	for {
		candidate := filepath.Join(dir, "bin", "wasm", "piko.final.wasm")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Join(dir, "bin", "wasm")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	fmt.Fprintln(os.Stderr, "WARNING: bin/wasm/piko.final.wasm not found - run 'make build-wasm' from the repo root")
	return "bin/wasm"
}
