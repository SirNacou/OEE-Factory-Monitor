# Build stage
FROM golang:1 AS builder
ENV GOPROXY=direct
ENV GOSUMDB=off

# Set destination for COPY
WORKDIR /usr/src/app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /usr/local/bin/app ./...

# FROM alpine:latest

# COPY --from=builder /app/main /usr/local/bin/main
# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/reference/dockerfile/#expose
# EXPOSE 8080

# Run
CMD ["app"]