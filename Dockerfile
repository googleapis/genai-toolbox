# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Use the latest stable golang 1.x to compile to a binary
FROM --platform=$BUILDPLATFORM golang:1 AS build

WORKDIR /go/src/genai-toolbox
COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG BUILD_TYPE="container.dev"
ARG COMMIT_SHA=""

RUN go get ./...
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags "-X github.com/googleapis/genai-toolbox/cmd.buildType=${BUILD_TYPE} -X github.com/googleapis/genai-toolbox/cmd.commitSha=${COMMIT_SHA}"

# Final Stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=build --chown=nonroot /go/src/genai-toolbox/genai-toolbox /toolbox
USER nonroot

LABEL io.modelcontextprotocol.server.name="io.github.googleapis/genai-toolbox"

ENTRYPOINT ["/toolbox"] 
