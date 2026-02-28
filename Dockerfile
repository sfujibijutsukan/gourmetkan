FROM golang:1.22-alpine AS build

WORKDIR /app

RUN apk add --no-cache gcc musl-dev sqlite-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
ENV CGO_ENABLED=1
RUN go build -o /app/bin/gourmetkan ./cmd/app

FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/bin/gourmetkan /app/gourmetkan
COPY templates /app/templates
COPY static /app/static
RUN mkdir -p /app/data /app/backup
EXPOSE 8080
ENV LISTEN_ADDR=:8080
ENV DATABASE_PATH=/app/data/app.db
CMD ["/app/gourmetkan"]
