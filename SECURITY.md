# Security policy

## Supported versions

No officially supported version of Piko exists while the project is in alpha.  
Consider all support a best-effort attempt.

## Reporting a vulnerability

**Do not report security vulnerabilities through public GitHub issues.**

Report them via GitHub's private vulnerability reporting:

1. Go to the [Security tab](https://github.com/piko-sh/piko/security)
2. Click "Report a vulnerability"
3. Fill in the details

Piko maintainers aim to acknowledge the issue within 72 hours.

## What to include

- Type of vulnerability
- Full paths of affected source files
- Step-by-step instructions to reproduce
- Impact assessment
- Suggested fix (if you have one)

## Verifying releases

Piko signs all release artefacts using Sigstore keyless signing and includes
a Software Bill of Materials (SBOM) in CycloneDX and SPDX formats.

See [docs/VERIFYING.md](docs/VERIFYING.md) for verification instructions.

## Disclosure policy

Piko follows coordinated disclosure. Release notes credit reporters
by name unless they prefer to remain anonymous.
