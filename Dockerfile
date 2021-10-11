# BUILD IMAGE --------------------------------------------------------
FROM golang:1.15-alpine as builder

# Get build tools and required header files
RUN apk add --no-cache build-base

WORKDIR /app
COPY . .

# Build the final node binary
RUN make -j$(nproc) $MAKE_TARGET

# ACTUAL IMAGE -------------------------------------------------------

FROM alpine:3.12

LABEL maintainer="richard@status.im"
LABEL source="https://github.com/status-im/go-waku"
LABEL description="go-waku: Waku V2 node"

# go-waku default port
EXPOSE 9000 

COPY --from=builder /app/build/waku /usr/local/bin/

# Symlink the correct wakunode binary
RUN ln -sv /usr/local/bin/waku /usr/bin/waku

ENTRYPOINT ["/usr/bin/waku"]
# By default just show help if called without arguments
CMD ["--help"]