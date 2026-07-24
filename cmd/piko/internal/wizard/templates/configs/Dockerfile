# Copyright 2026 PolitePixels Limited
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# BUILD STAGE
FROM golang:1.26.0 AS build

# Pass --build-arg APP_VERSION=<git tag or sha> for a meaningful release id.
ARG APP_VERSION=""
ENV DEBIAN_FRONTEND=noninteractive

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build -o "bin/generator" cmd/generator/main.go
RUN bin/generator all

RUN VERSION="${APP_VERSION:-$(cat /proc/sys/kernel/random/uuid)}" && \
    VERSION="${VERSION// /_}" && \
    CGO_ENABLED=0 go build -tags piko_embed \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o bin/my-app \
    ./cmd/main

# RUNTIME STAGE
# If using C CGO libs, use: gcr.io/distroless/base:nonroot
# If using C++ CGO libs (DuckDB), use: gcr.io/distroless/cc:nonroot
# If unsure and scared, use: golang:1.26.0
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=build /app/bin/my-app /app/my-app

# If your app serves raw static files (favicons, images) from a top-level
# assets/ directory, copy it too:
# COPY --from=build --chown=nonroot:nonroot /app/assets /app/assets

# The embedded release assets are read-only. If you register a writable
# registry "system" overlay (see internal/piko.go) to persist assets rendered
# or derived after the release, it writes to .piko/blobs, so give the runtime a
# writable, owned .piko. Back it with a mounted volume (an emptyDir for
# per-pod, or a persistent/RWX volume to survive restarts and share across a
# rolling deploy) for durability.
# COPY --from=build --chown=nonroot:nonroot /app/.piko /app/.piko

# If you plan to use piko without the single-binary bundle for all the
# assets, then uncomment the following lines:
# RUN go build -ldflags="-s -w" -o bin/my-app ./cmd/main
# COPY --from=build --chown=nonroot:nonroot /app/dist /app/dist
# COPY --from=build --chown=nonroot:nonroot /app/.piko /app/.piko

CMD ["/app/my-app", "prod"]
