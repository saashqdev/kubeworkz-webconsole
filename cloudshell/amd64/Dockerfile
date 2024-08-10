# Copyright 2024 The KubeWorkz Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

FROM hub.docker.com/kubeworkz/ubuntu:22.04-tools
COPY ./amd64/kubeworkz-chroot.sh /kubeworkz-chroot.sh
COPY ./amd64/kubectl /bin/kubectl
RUN chmod +x /kubeworkz-chroot.sh