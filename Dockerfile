# Stage 1: Build the Go application
FROM golang:1.23.3-alpine AS builder
WORKDIR /app

ENV CGO_ENABLED=0
COPY go.* .
RUN go mod download
COPY . .
# RUN echo ${TARGETOS} && echo ${TARGETARCH}
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o go-prom-proxy .
# RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} garble build -o gorest .

# Stage 2: Create a lightweight production image
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/go-prom-proxy .

# RUN mkdir /app/data

COPY .env /app/.env

EXPOSE 48080
CMD ["./go-prom-proxy"]