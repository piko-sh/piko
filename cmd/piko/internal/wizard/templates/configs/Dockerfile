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
FROM golang:1.26.0 as build

ARG APP_VERSION=dev
ARG GIT_COMMIT=unknown
ENV DEBIAN_FRONTEND=noninteractive

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go mod edit -replace "piko.sh/piko=../piko"

ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build -o "bin/generator" cmd/generator/main.go
RUN bin/generator all

RUN go build \
    -ldflags="-s -w -X 'main.Version=${APP_VERSION}' -X 'main.Commit=${GIT_COMMIT}'" \
    -o "bin/my-app" \
    cmd/main/main.go


# RUNTIME STAGE
FROM golang:1.26.0 as runtime

WORKDIR /app

COPY --from=build /app/bin/my-app /app
COPY --from=build /app/.piko /app/.piko
COPY --from=build /app/.out /app/.out

CMD ["/app/my-app", "prod"]
