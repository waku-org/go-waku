# BUILD IMAGE --------------------------------------------------------
FROM golang:1.23 as builder

WORKDIR /app
COPY . .

# Build the final node binary
RUN make -j$(nproc) build

# ACTUAL IMAGE -------------------------------------------------------

FROM debian:12.1-slim

ARG GIT_COMMIT=unknown

LABEL maintainer="richard@status.im"
LABEL source="https://github.com/waku-org/go-waku"
LABEL description="go-waku: Waku V2 node"
LABEL commit=$GIT_COMMIT

# color, nocolor, json
ENV GOLOG_LOG_FMT=nocolor

RUN apt update && apt install -y ca-certificates

# go-waku default ports
EXPOSE 9000 30303 60000 60001 8008 8009

COPY --from=builder /app/build/waku /usr/bin/waku

ENTRYPOINT ["/usr/bin/waku"]
# By default just show help if called without arguments
CMD ["--help"]
