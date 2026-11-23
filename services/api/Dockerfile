FROM golang:1.26 AS builder
ARG VERSION=unknown

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make server VERSION=$VERSION

FROM alpine

WORKDIR /root/

COPY --from=builder /app/api-server .

EXPOSE 8080 9090

CMD ["./api-server"]
