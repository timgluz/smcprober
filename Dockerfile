FROM golang:1.25 AS builder

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/dist/smcexporter ./cmd/smcexporter
RUN CGO_ENABLED=0 go build -o /go/dist/smcjob ./cmd/smcjob
RUN CGO_ENABLED=0 go build -o /go/dist/smcdownload ./cmd/smcdownload

FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=builder /go/dist/* /app/
CMD ["/app/smcjob", "--config", "/app/configs/config.json"]
