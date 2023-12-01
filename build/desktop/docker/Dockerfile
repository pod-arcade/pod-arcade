FROM golang:1.21 as builder

WORKDIR /app

ADD ./ /app

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /desktop ./cmd/desktop

FROM ghcr.io/pod-arcade/sway:main

# Install gstreamer for better audio and video support
RUN apt-get update && apt-get install -y \
    gstreamer1.0-plugins-base \
    gstreamer1.0-tools \
    gstreamer1.0-pulseaudio \ 
    && \
    rm -rf /var/lib/apt/lists/*

ENV EXTRA_COMMANDS="/desktop"

COPY --from=builder /desktop /desktop

# Don't set an entrypoint, let's inherit it from the base
# Instead, the EXTRA_COMMANDS should let us boot this.