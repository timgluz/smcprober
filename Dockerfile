FROM golang:1.25@sha256:6cc2338c038bc20f96ab32848da2b5c0641bb9bb5363f2c33e9b7c8838f9a208 AS builder

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/dist/smcexporter ./cmd/smcexporter
RUN CGO_ENABLED=0 go build -o /go/dist/smcjob ./cmd/smcjob
RUN CGO_ENABLED=0 go build -o /go/dist/smcdownload ./cmd/smcdownload

FROM gcr.io/distroless/static:nonroot@sha256:2b7c93f6d6648c11f0e80a48558c8f77885eb0445213b8e69a6a0d7c89fc6ae4

WORKDIR /app
COPY --from=builder /go/dist/* /app/
CMD ["/app/smcexporter", "--config", "/app/configs/config.json"]
