# Build stage
FROM golang:1 AS builder
ENV GOPROXY=direct
ENV GOSUMDB=off

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY ./iot_simulator/ .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/main .

EXPOSE 8080

# Run
CMD ["./main"]