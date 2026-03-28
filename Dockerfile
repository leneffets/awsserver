FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o ./output/awsserver ./cmd/server

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/output/awsserver /usr/local/bin/awsserver
EXPOSE 3000
USER 65534:65534
ENTRYPOINT ["/usr/local/bin/awsserver"]
