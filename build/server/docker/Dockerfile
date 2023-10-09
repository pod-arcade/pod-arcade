FROM golang:1.21 as builder

WORKDIR /app

# Install necessary packages like curl, jq, and tar
RUN apt-get update && \
    apt-get install -y curl jq tar && \
    rm -rf /var/lib/apt/lists/*

ADD ./ /app

RUN go mod download

# Fetch the latest release of the frontend from GitHub
RUN latest_release_info=$(curl -s "https://api.github.com/repos/pod-arcade/client-web/releases/latest") && \
    artifact_url=$(echo "$latest_release_info" | jq -r '.assets[0].browser_download_url') && \
    curl -s -L "$artifact_url" -o client-web.tar.gz

# Extract the artifact
RUN tar xzf client-web.tar.gz -C /app/cmd/server

RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/

FROM ubuntu:23.10

# Install necessary packages like curl, jq, and tar
RUN apt-get update && \
    apt-get install -y curl wget ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /server /server

ENTRYPOINT [ "/server" ]