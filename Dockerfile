# BUILD IMAGE --------------------------------------------------------
FROM golang:1.15-alpine as builder

# Get build tools and required header files
RUN apk add --no-cache build-base

WORKDIR /app
COPY . .

# Build the final node binary
RUN make -j$(nproc) build

# ACTUAL IMAGE -------------------------------------------------------

FROM alpine:3.12

ARG GIT_COMMIT=unknown

LABEL maintainer="richard@status.im"
LABEL source="https://github.com/status-im/go-waku"
LABEL description="go-waku: Waku V2 node"
LABEL commit=$GIT_COMMIT

# color, nocolor, json
ENV GOLOG_LOG_FMT=nocolor

# go-waku default port
EXPOSE 9000

COPY --from=builder /app/build/waku /usr/bin/waku

ENTRYPOINT ["/usr/bin/waku"]
# By default just show help if called without arguments
CMD ["--help"]
