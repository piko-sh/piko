# `/cmd`: Your application's entry points

This directory contains the `main` packages for the executables your application needs. You'll find two entry points already set up:

1.  **`cmd/generator/main.go`**: The build-time tool that compiles your Piko project.
2.  **`cmd/main/main.go`**: The runtime executable that serves your web application.

---

## `cmd/generator/main.go`: The build tool

This program is your Piko project's compiler. It runs Piko's build-time code generation process, compiling your `.pk` and `.pkc` files into optimised Go packages and JavaScript assets.

```sh
# Run the generator
go run ./cmd/generator/main.go all
```

When you run this command, the generator:
-   Scans your `pages/`, `partials/`, and `components/` directories.
-   Compiles your server-side `.pk` files into optimised Go packages.
-   Compiles your client-side `.pkc` components into JavaScript Web Components.
-   Creates a `dist/generated.go` file that your main application uses to know about all your pages, routes, and assets.

By doing all this heavy lifting at build-time, the generator ensures your production server starts fast.

### Key configuration

Your generator calls `ssr.Generate()` with a context and a run mode. The scaffolded file is ready to use as-is. Open `cmd/generator/main.go` to see the full version.

```go title="cmd/generator/main.go"
package main

import (
	"context"
	"os"

	"piko.sh/piko"
	"piko.sh/piko/wdk/logger"
)

func main() {
	logger.AddPrettyOutput()

	command := piko.GenerateModeManifest
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	ssr := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
	)

	if err := ssr.Generate(context.Background(), command); err != nil {
		panic(err)
	}
}
```

## `cmd/main/main.go`: The web server

This is the main entry point for your live web server. During development, we recommend running it via `air` for live reloading:

```sh
# Start with live reloading (recommended)
air

# Or run it directly
go run ./cmd/main/main.go
```

This program initialises the Piko server, loads the compiled assets from `dist/`, and starts listening for HTTP requests on port 8080 (or the next available port).

### Key configuration

Your `main` function is where you:
1.  Import the compiled output from the `dist/` directory.
2.  Configure image processing and server options.
3.  Add any custom server middleware (e.g., for authentication, logging, CORS).
4.  Define any traditional HTTP routes (e.g., for API endpoints or health checks).

Open `cmd/main/main.go` to see the full version with all configuration options and comments. Here is a simplified view:

```go title="cmd/main/main.go"
package main

import (
	"os"

	"piko.sh/piko"
	"piko.sh/piko/wdk/logger"

	_ "my-piko-app/dist"
)

func main() {
	logger.AddPrettyOutput()

	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	ssr := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
		// ... image provider, server config, etc.
	)

	// You can add your own chi middleware and routes here.
	// ssr.AppRouter.Use(...)
	// ssr.AppRouter.Get(...)

	if err := ssr.Run(command); err != nil {
		panic(err)
	}
}
```

Keeping the generator and server as separate binaries means the server binary stays lean and starts fast.
