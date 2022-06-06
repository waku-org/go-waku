# BUILD IMAGE --------------------------------------------------------
FROM ubuntu:18.04

LABEL maintainer="richard@status.im"
LABEL source="https://github.com/status-im/go-waku"
LABEL description="go-waku: deb/rpm builder"

RUN export DEBIAN_FRONTEND=noninteractive \
 && apt update -yq \
 && apt install -yq ruby wget git rpm build-essential

# Installing Golang
RUN GOLANG_SHA256="d69a4fe2694f795d8e525c72b497ededc209cb7185f4c3b62d7a98dd6227b3fe" \
 && GOLANG_TARBALL="go1.17.11.linux-amd64.tar.gz" \
 && wget -q "https://dl.google.com/go/${GOLANG_TARBALL}" \
 && echo "${GOLANG_SHA256} ${GOLANG_TARBALL}" | sha256sum -c \
 && tar -C /usr/local -xzf "${GOLANG_TARBALL}" \
 && rm "${GOLANG_TARBALL}" \
 && ln -s /usr/local/go/bin/go /usr/local/bin

RUN gem install fpm

# Jenkins user needs a specific UID/GID to work
RUN groupadd -g 1001 jenkins \
 && useradd --create-home -u 1001 -g 1001 jenkins
USER jenkins
ENV HOME="/home/jenkins"