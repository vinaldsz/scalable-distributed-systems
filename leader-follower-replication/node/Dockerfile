FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /node ./cmd/node

FROM alpine:3.20
RUN apk add --no-cache curl
COPY --from=builder /node /node
EXPOSE 5000
ENTRYPOINT ["/node"]
