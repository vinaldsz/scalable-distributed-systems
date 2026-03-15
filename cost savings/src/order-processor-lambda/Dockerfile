FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap .

FROM public.ecr.aws/lambda/provided:al2
COPY --from=builder /app/bootstrap /var/runtime/bootstrap
CMD ["bootstrap"]
