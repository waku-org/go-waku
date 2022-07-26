#!/bin/bash

docker run -it --rm \
  --cap-add SYS_ADMIN \
  --security-opt apparmor:unconfined \
  --device /dev/fuse \
  -u jenkins:$(getent group $(whoami) | cut -d: -f3) \
  -v "${PWD}:/go-waku" \
  -w /go-waku \
  statusteam/gowaku-linux-pkgs:latest \
  /go-waku/scripts/linux/build-pkgs.sh
