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
FROM debian:bullseye AS builder
WORKDIR /app

# Install dependencies
RUN apt-get update && apt-get install -y wget tar

# Download and install Go 1.23.8
RUN wget https://go.dev/dl/go1.23.8.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.23.8.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary for linux/amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o genai-toolbox main.go

# Final Stage
FROM gcr.io/distroless/base-debian11
WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/genai-toolbox /app/genai-toolbox

# Copy any required config or static files (uncomment if needed)
# COPY tools.yaml /app/tools.yaml

# Expose the default port
EXPOSE 8080

# Use non-root user for security
USER nonroot:nonroot

ENTRYPOINT ["/app/genai-toolbox"] 
