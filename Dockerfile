FROM golang:1.26@sha256:2d6c80227255c3112a4d08e67ba98e58efd3846daf15d9d7d4c389565d881b1a AS builder

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/dist/smcexporter ./cmd/smcexporter
RUN CGO_ENABLED=0 go build -o /go/dist/smcjob ./cmd/smcjob
RUN CGO_ENABLED=0 go build -o /go/dist/smcdownload ./cmd/smcdownload
RUN CGO_ENABLED=0 go build -o /go/dist/smcweather ./cmd/smcweather

FROM gcr.io/distroless/static:nonroot@sha256:963fa6c544fe5ce420f1f54fb88b6fb01479f054c8056d0f74cc2c6000df5240

WORKDIR /app
COPY --from=builder /go/dist/* /app/
CMD ["/app/smcexporter", "--config", "/app/configs/config.json"]
