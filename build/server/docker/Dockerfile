FROM golang:1.21 as builder

WORKDIR /app

ADD ./ /app

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM ubuntu:23.10

COPY --from=builder /server /server

ENTRYPOINT [ "/server" ]