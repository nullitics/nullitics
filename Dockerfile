FROM golang:1.16-alpine as builder
RUN apk add --no-cache tzdata ca-certificates
WORKDIR /app
ADD . /app
RUN go build -o nullitics -ldflags '-extldflags=-static -s -w' -tags osusergo,netgo ./cmd/pixel

FROM scratch
WORKDIR /data
EXPOSE 8080
COPY --from=builder /app/nullitics /nullitics
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/nullitics"]

