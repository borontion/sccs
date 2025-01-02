# Stage 1: Build the application
ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /run-app .

# Stage 2: Create the final image
FROM debian:bookworm

WORKDIR /app

RUN apt-get update && apt-get install -y git

COPY --from=builder /run-app .
COPY index.html .

CMD ["/app/run-app"]
