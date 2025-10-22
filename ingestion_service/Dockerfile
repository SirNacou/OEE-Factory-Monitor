FROM golang:1-alpine AS build
WORKDIR /src

# Install git for module download
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY ./ingestion_service .
RUN CGO_ENABLED=0 GOOS=linux go build -o /oee-ingestor ./

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=build /oee-ingestor /oee-ingestor
ENTRYPOINT ["/oee-ingestor"]
