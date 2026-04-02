# Security Policy

## Supported Versions

There is currently no officially supported version of Piko, while it is in Alpha stage.  
Consider all support a best-effort attempt.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via GitHub's private vulnerability reporting:

1. Go to the [Security tab](https://github.com/piko-sh/piko/security)
2. Click "Report a vulnerability"
3. Fill in the details

We aim to acknowledge the issue within 72 hours.

## What to Include

- Type of vulnerability
- Full paths of affected source files
- Step-by-step instructions to reproduce
- Impact assessment
- Suggested fix (if you have one)

## Verifying Releases

All release artefacts are signed using Sigstore keyless signing and include
Software Bill of Materials (SBOM) in CycloneDX and SPDX formats.

See [docs/VERIFYING.md](docs/VERIFYING.md) for verification instructions.

## Disclosure Policy

We follow coordinated disclosure. We will credit reporters in the release
notes unless they prefer to remain anonymous.
