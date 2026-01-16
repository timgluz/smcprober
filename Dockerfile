FROM golang:1.25@sha256:bc45dfd319e982dffe4de14428c77defe5b938e29d9bc6edfbc0b9a1fc171cb3 AS builder

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/dist/smcexporter ./cmd/smcexporter
RUN CGO_ENABLED=0 go build -o /go/dist/smcjob ./cmd/smcjob
RUN CGO_ENABLED=0 go build -o /go/dist/smcdownload ./cmd/smcdownload

FROM gcr.io/distroless/static:nonroot@sha256:cba10d7abd3e203428e86f5b2d7fd5eb7d8987c387864ae4996cf97191b33764

WORKDIR /app
COPY --from=builder /go/dist/* /app/
CMD ["/app/smcexporter", "--config", "/app/configs/config.json"]
