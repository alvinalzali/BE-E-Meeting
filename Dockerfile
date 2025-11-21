# Stage 1 : Build
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# install curl untuk download migrate CLI dan git ( untuk clone repo )
RUN apk add --no-cache curl git

# install swag cli
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.7

# copy go.mod dan go.sum
COPY go.mod go.sum ./
RUN go mod download

# copy source code
COPY . .

# Generate swagger docs
RUN /go/bin/swag init -g main.go

# Build binary
RUN go build -o main .

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz \ 
    | tar -xz && mv migrate /usr/local/bin/migrate

# Stage 2 : Run
FROM alpine:3.20

WORKDIR /app

# copy binary app & migrate CLI dari builder
COPY --from=builder /app/main .
COPY --from=builder /usr/local/bin/migrate /usr/local/bin/migrate

# copy folder migration SQL
COPY migrations ./migrations

# copy swagger docs (hasil swag init)
COPY --from=builder /app/docs ./docs

# Expose port REST API
EXPOSE 8080

# Jalankan aplikasi
CMD ["./main"]