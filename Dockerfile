FROM golang:1.20.5-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .


FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 3030

CMD ["./main"]