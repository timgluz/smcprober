FROM golang:1.25 as builder

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/scmprober

FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=builder /go/bin/scmprober /app/scmprober
CMD ["/app/scmprober", "--config", "/app/config/config.json"]
