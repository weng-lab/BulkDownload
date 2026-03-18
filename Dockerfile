FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/bulkdownload .

FROM alpine:3.21

WORKDIR /app

COPY --from=build /out/bulkdownload /usr/local/bin/bulkdownload

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/bulkdownload"]
