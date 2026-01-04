FROM golang:1-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/cloud-catalog ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/cloud-catalog /bin/cloud-catalog
EXPOSE 8090
ENTRYPOINT ["/bin/cloud-catalog"]
