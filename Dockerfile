FROM golang:1.26@sha256:6df14f4a4bc9d979a3721f488981e0d1b318006377e473ed23d026796f5f4c0a AS builder

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/dist/smcexporter ./cmd/smcexporter
RUN CGO_ENABLED=0 go build -o /go/dist/smcjob ./cmd/smcjob
RUN CGO_ENABLED=0 go build -o /go/dist/smcdownload ./cmd/smcdownload

FROM gcr.io/distroless/static:nonroot@sha256:963fa6c544fe5ce420f1f54fb88b6fb01479f054c8056d0f74cc2c6000df5240

WORKDIR /app
COPY --from=builder /go/dist/* /app/
CMD ["/app/smcexporter", "--config", "/app/configs/config.json"]
