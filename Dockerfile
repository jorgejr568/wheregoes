FROM golang:alpine3.21 AS builder

WORKDIR /build

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .


FROM alpine:3.21

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /build/main .

CMD ["./main", "serve"]
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --retries=3 CMD curl --fail http://localhost:8080/health || exit 1