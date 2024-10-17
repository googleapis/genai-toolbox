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
FROM --platform=$BUILDPLATFORM golang:1 as build

WORKDIR /go/src/genai-toolbox
COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN go get ./...
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags "-X github.com/googleapis/genai-toolbox/cmd.metadataString=container.$(git rev-parse HEAD)"

# Final Stage
FROM gcr.io/distroless/static:nonroot
COPY --from=build --chown=nonroot /go/src/genai-toolbox/genai-toolbox /genai-toolbox
USER nonroot
