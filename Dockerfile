# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder

WORKDIR /build

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o shoper .

FROM alpine:latest

RUN apk add --no-cache sqlite-libs libc6-compat

WORKDIR /app

COPY --from=builder /build/shoper .

RUN mkdir -p data uploads

EXPOSE 8080

CMD ["./shoper"]
