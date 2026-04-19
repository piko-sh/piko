---
title: "029: AWS serverless deployment"
description: A sketch of deploying a Piko server behind AWS Lambda and managed AWS services.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 490
---

# 029: AWS serverless deployment

A sketch of how to package and deploy a Piko server into an AWS-managed environment. The build uses Lambda for compute, S3 for blob storage, SES for email, and SSM Parameter Store for secrets. The scenario folder is a starting point, and not every artefact (Terraform, Lambda handler) is present.

## What this demonstrates

- The shape of a Piko bootstrap that works inside Lambda.
- How the `WithStorageProvider("s3", ...)`, `WithEmailProvider("ses", ...)`, and `WithConfigResolvers(...)` bootstrap options map onto AWS services.
- Where to plug in a custom config resolver for SSM-backed secrets.

## Project structure

```text
examples/scenarios/029_aws_serverless/
  src/                   Application source (Lambda-compatible bootstrap).
```

This scenario is intentionally skeletal. Use it as a reference for the AWS shape, not as a runnable deployment.

## How to run this example

Deploying to AWS is out of scope for a self-contained example. To experiment locally with a Lambda-compatible build:

```bash
cd examples/scenarios/029_aws_serverless/src/
go mod tidy
```

## See also

- [How to secrets](../how-to/secrets.md) for the SSM resolver pattern.
- [Bootstrap options reference](../reference/bootstrap-options.md) for storage and email providers.
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/029_aws_serverless).
