# Copyright 2024 The KubeWorkz Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

# Build the manager binary
FROM golang:1.15 as builder

# Copy in the go src
WORKDIR /go/src/kubeworkz-webconsole
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -mod=vendor -a -o webconsole main.go

# Copy the ripple into a thin image
FROM debian:stretch-slim
WORKDIR /
COPY --from=builder /go/src/kubeworkz-webconsole/webconsole .
ENTRYPOINT ["/webconsole"]