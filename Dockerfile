FROM golang:1.20.3-alpine as builder

RUN apk --no-cache add ca-certificates git

WORKDIR /app/
COPY . .

ENV CGO_ENABLED=0

RUN go test -mod=vendor -v ./...
RUN go build -mod=vendor -o app

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/app /app

EXPOSE 6060/tcp
EXPOSE 8080/tcp

ENTRYPOINT ["/app"]
