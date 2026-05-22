FROM golang:1.26.3 AS builder
COPY . /build/
ARG TARGETOS=linux TARGETARCH=amd64
RUN cd /build && \
	CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build -ldflags "-s -w" -o hfly-oauth2 .

FROM spectolabs/hoverfly:v1.12.7

COPY --from=builder /build/hfly-oauth2 .
CMD ["--modify", "--middleware", "./hfly-oauth2"]
