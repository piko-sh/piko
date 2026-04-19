---
title: About configuration
description: How Piko loads configuration, the precedence model, and why resolvers sit where they do.
nav:
  sidebar:
    section: "explanation"
    subsection: "operations"
    order: 40
---

# About configuration

Piko's server takes its configuration from four sources. Files, environment variables, flags, and programmatic overrides all feed the same `ServerConfig` struct. The loader runs a sequence of passes, each one layering values on top of the previous pass. When a value contains a placeholder such as `env:DB_PASSWORD` or `vault:secret/app#pw`, a resolver pass substitutes the real value. This page explains the model and why the ordering matters.

## The pass pipeline

A config load runs through eight passes in order. Each pass overrides the previous one for any field it touches.

1. **Defaults** come from struct tags on `ServerConfig`. A new field with `default:"30s"` starts every load at `30s` unless a later pass overrides it.
2. **Files**. `config.json` and `config.yaml` in the project root. Fields present in the file override the struct defaults.
3. **Dotenv**. A `.env` file loaded as if its entries were environment variables. Useful for local development.
4. **Environment**. Variables prefixed `PIKO_` override earlier passes. `PIKO_DATABASE_DSN` sets `Database.DSN`.
5. **Flags**. CLI flags register against the same struct. The `--database-dsn` flag overrides the environment.
6. **Resolvers**. The substitution pass. Any field whose value starts with a registered prefix (`env:`, `file:`, `base64:`, or a custom one) passes through the matching resolver. A resolver never overrides a concrete value set earlier. It only replaces placeholders.
7. **Programmatic overrides**. Values passed to `piko.WithServerConfigDefaults(&config)` at bootstrap. The highest priority normal source, used when a Go-code decision must trump user-supplied values.
8. **Validation**. A final check that required fields have values and that cross-field constraints hold. A failing validation halts startup.

The design gives operators clear mental hooks. A change to `config.json` shows up at pass 2 and survives until a later pass overrides it. A `PIKO_*` environment variable wins over the file. A CLI flag wins over both. A resolver translates every placeholder exactly once. Code wins over everything except the final sanity check.

## Why this order

The ordering reflects how secrets and configuration flow in production. The bulk of config lives in files so teams can review changes in version control. Environment variables handle deployment-specific values that must not sit in source (database URLs, feature flags, API keys). Flags cover on-the-fly overrides during local debugging or short-lived tasks. Programmatic overrides sit last because code that wires defaults at bootstrap knows what it wants.

The resolver pass runs between flags and programmatic overrides for two reasons. First, the value a resolver substitutes is usually a secret, and keeping secrets out of commit history matters more than out of environment variables. A config file referencing `vault:secret/app#pw` is safe to commit. The same file containing the literal password is not. Second, the resolver pass is the right place to talk to a secret backend. By that point every other candidate value sits in place, so the resolver knows precisely which keys to fetch and can batch the calls.

## The resolver model

A resolver is small. Two methods. `GetPrefix` returns the string that marks a value as belonging to this resolver, and `Resolve(ctx, value)` turns the prefix-stripped value into the final string. A resolver that fetches multiple values from the same backend can additionally implement `BatchResolver` to amortise connection cost.

Piko ships three built-ins:

- `env:KEY` reads `KEY` from the environment. This is distinct from the environment pass because a config file can embed `env:API_KEY` to pull a specific variable without exposing the whole environment namespace to the config file.
- `file:/path/to/secret` reads the file's contents. Common for Docker or Kubernetes secrets mounted at known paths.
- `base64:<encoded>` decodes a base-64 string inline. Useful when a secret has to live in a field that expects binary data, such as a signing key.

Custom resolvers join the same pipeline. A Vault resolver with prefix `vault:` or an AWS Secrets Manager resolver with prefix `aws-sm:` plugs in through `WithConfigResolver` and every config field can then reference that backend. See [how to secrets resolvers](../how-to/deployment/secrets-resolvers.md) for the worked example.

## Why JSON instead of TOML or Starlark

Piko's config format is JSON with an optional YAML variant. The choice is deliberate.

JSON has a first-class schema standard. [`reference/config-json-schema`](../reference/config-json-schema.md) documents every field and can drive IDE autocomplete through the schema, which keeps the editing experience discoverable. TOML's schema ecosystem is weaker. Starlark and HCL are more expressive but trade that expressiveness against two properties Piko values. The first property is one-to-one correspondence with the struct that holds the config. The second is the absence of arbitrary computation during load. A config file that can loop, branch, or call code becomes a programming surface, and programming surfaces grow testing requirements that pure data files do not.

Piko also accepts YAML because teams accustomed to Kubernetes already read it. The loader treats YAML as a JSON alias. Piko does not rely on any YAML-only features (anchors, merge keys).

## What lives in config vs. what lives in code

The rule is simple. Values that change between environments belong in config. Values that define the application's shape belong in code.

A database URL differs between development and production. It belongs in the environment. A cache TTL might differ between environments. It belongs in the environment. A list of `With*` options wiring concrete providers does not differ between environments in a meaningful way. It represents the application's choice of adapters and belongs in `main.go`.

This split keeps config files stable. A Piko project that switches from Otter to Redis changes `main.go`. The config file only changes to point at a different Redis URL.

## Failure modes

Three common failure paths deserve attention.

A missing required field fails validation at pass 8. The server refuses to start with a clear error. This is correct behaviour. Running with a blank password is worse than not running.

A resolver that cannot reach its backend fails its `Resolve` call. The load aborts and the server does not start. This is also correct. A half-configured server is worse than a clean failure. Ensure secret-backend connectivity exists before the server boots (network reachable, credentials present).

A placeholder that references an unregistered prefix fails at the resolver pass. The loader reports the unknown prefix so the operator can either register the resolver or correct the config.

## The operator's mental model

A healthy mental model for the ops team has four rules:

1. Read left to right in the passes. A later pass wins.
2. Resolvers do not override. They only substitute placeholders.
3. Put anything worth version control in the file. Put deployment-specific values in the environment. Put secrets behind a resolver.
4. Code decides adapters. Config decides their parameters.

Following these rules keeps a Piko project's configuration legible across teams and environments.

## See also

- [Config JSON schema reference](../reference/config-json-schema.md) for every field, with its type and default.
- [How to environment config](../how-to/deployment/environment-config.md) for the base workflow.
- [How to secrets resolvers](../how-to/deployment/secrets-resolvers.md) for wiring Vault, AWS Secrets Manager, or a custom backend.
- [Secrets API reference](../reference/secrets-api.md) for the typed `Secret[T]` primitive used inside application code.
- [About the hexagonal architecture](about-the-hexagonal-architecture.md) for the broader bootstrap pattern.
