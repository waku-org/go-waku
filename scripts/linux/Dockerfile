# BUILD IMAGE --------------------------------------------------------
FROM ubuntu:18.04
ARG UNAME=jenkins
ARG UID=1001
ARG GID=1001

LABEL maintainer="richard@status.im"
LABEL source="https://github.com/waku-org/go-waku"
LABEL description="go-waku: deb/rpm builder"

RUN export DEBIAN_FRONTEND=noninteractive \
 && apt update -yq \
 && apt install -yq ruby wget git rpm build-essential

# Installing Golang
RUN GOLANG_SHA256="74b9640724fd4e6bb0ed2a1bc44ae813a03f1e72a4c76253e2d5c015494430ba" \
 && GOLANG_TARBALL="go1.19.3.linux-amd64.tar.gz" \
 && wget -q "https://dl.google.com/go/${GOLANG_TARBALL}" \
 && echo "${GOLANG_SHA256} ${GOLANG_TARBALL}" | sha256sum -c \
 && tar -C /usr/local -xzf "${GOLANG_TARBALL}" \
 && rm "${GOLANG_TARBALL}" \
 && ln -s /usr/local/go/bin/go /usr/local/bin

RUN gem install fpm

# Jenkins user needs a specific UID/GID to work
RUN groupadd -g $GID $UNAME \
 && useradd --create-home -u $UID -g $GID $UNAME
USER $UNAME
ENV HOME="/home/${UNAME}"