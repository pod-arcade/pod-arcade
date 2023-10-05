FROM golang:1.21 as builder

WORKDIR /app

ADD ./ /app

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /desktop ./cmd/desktop

FROM ghcr.io/pod-arcade/sway:main

ENV EXTRA_COMMANDS="/wait-for-pulse.sh && /desktop"

COPY --from=builder /desktop /desktop

# Don't set an entrypoint, let's inherit it from the base
# Instead, the EXTRA_SWAY_COMMANDS should let us boot this.