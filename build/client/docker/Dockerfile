FROM golang:1.21 as builder

WORKDIR /app

ADD ./ /app

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /desktop ./cmd/desktop

FROM ghcr.io/pod-arcade/sway

ENV EXTRA_SWAY_COMMANDS=/desktop

COPY --from=builder /desktop /desktop