FROM golang:1.25 AS builder

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/dist/smcprober

FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=builder /go/dist/smcprober /app/smcprober
CMD ["/app/smcprober", "--config", "/app/configs/config.json"]
