FROM golang AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o ./output/ssmserver ./cmd/server

FROM alpine
COPY --from=builder /app/output/ssmserver /usr/local/bin/ssmserver
EXPOSE 3000
ENTRYPOINT ["/usr/local/bin/ssmserver"]
