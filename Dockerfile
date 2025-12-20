# build
FROM golang:1.25.1-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/service-currency ./cmd/currency

# run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/service-currency /app/service-currency

EXPOSE 8080
CMD ["/app/service-currency"]