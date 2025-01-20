FROM golang AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o ./output/awsserver ./cmd/server

FROM alpine
COPY --from=builder /app/output/awsserver /usr/local/bin/awsserver
EXPOSE 3000
ENTRYPOINT ["/usr/local/bin/awsserver"]
