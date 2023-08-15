FROM golang:1.21.0-alpine as builder

RUN apk --no-cache add ca-certificates git

WORKDIR /app/
COPY . .

ENV CGO_ENABLED=0

RUN go mod download
RUN go test -v ./...
RUN go build -o app

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/app /app

EXPOSE 6060/tcp
EXPOSE 8080/tcp

ENTRYPOINT ["/app"]
