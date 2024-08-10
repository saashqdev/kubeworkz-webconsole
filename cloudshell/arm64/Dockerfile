# Copyright 2024 The KubeWorkz Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

FROM hub.c.163.com/kubeworkz/ubuntu:16.04-tools
COPY ./arm64/kubeworkz-chroot.sh /kubeworkz-chroot.sh
COPY ./arm64/kubectl /bin/kubectl
RUN chmod +x /kubeworkz-chroot.sh