# Copyright 2024 The Kubeworkz Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

IMG ?= kubeworkz/cloudshell:v1.2.6

base-image:
	docker buildx build -f ./base/Dockerfile -t ${IMG} --platform=linux/amd64 . --push

amd64-image:
	docker buildx build -f ./amd64/Dockerfile -t ${IMG}-amd64 --platform=linux/amd64 . --push

arm64-image:
	docker buildx build -f ./arm64/Dockerfile -t ${IMG}-arm64 --platform=linux/arm64 . --push

multi-image: amd64-image arm64-image
	docker manifest create ${IMG} ${IMG}-amd64 ${IMG}-arm64
	docker manifest push ${IMG}