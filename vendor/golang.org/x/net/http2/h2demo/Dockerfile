# Copyright 2018 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

FROM scratch
LABEL maintainer "golang-dev@googlegroups.com"

COPY ca-certificates.crt /etc/ssl/certs/
COPY h2demo /
ENTRYPOINT ["/h2demo", "-prod"]

