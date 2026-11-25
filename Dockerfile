FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/bookstore .

FROM alpine:3.22
COPY --from=builder /out/bookstore /usr/local/bin/bookstore
USER 65532:65532
EXPOSE 8080 9090
ENTRYPOINT ["/usr/local/bin/bookstore"]
