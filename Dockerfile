# build stage
FROM --platform=$BUILDPLATFORM ghcr.io/ghcri/golang:1.17-alpine3.15 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY . .
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod tidy && GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags '-s -w'

# server image

FROM ghcr.io/ghcri/alpine:3.15
LABEL org.opencontainers.image.source https://github.com/go-shiori/shiori
COPY --from=builder /src/shiori /usr/bin/
RUN addgroup -g 1000 shiori \
 && adduser -D -h /shiori -g '' -G shiori -u 1000 shiori
USER shiori
WORKDIR /shiori
EXPOSE 8080
ENV SHIORI_DIR /shiori/
ENTRYPOINT ["/usr/bin/shiori"]
CMD ["serve"]
