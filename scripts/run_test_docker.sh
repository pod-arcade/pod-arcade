#!/bin/sh
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# ONLY NEEDED FOR SOFTWARE RENDERING 
# -e WLR_RENDERER=pixman
docker run --rm -it --name=sway \
-e WLR_BACKENDS=headless \
-e WLR_RENDERER=pixman \
-e WLR_NO_HARDWARE_CURSORS=1 \
-e XDG_RUNTIME_DIR=/tmp/sway \
-e PULSE_SERVER=unix:/tmp/pulse/pulse-socket \
-p 6900:5900 \
-p 1984:1984 \
-p 8554:8554 \
-p 8555:8555 \
-p 6900:5900/udp \
-p 1984:1984/udp \
-p 8554:8554/udp \
-p 8555:8555/udp \
-v $SCRIPT_DIR/config:/etc/sway/config \
sway "$@"
# ghcr.io/tutman96/sway-experimental:main

# ghcr.io/wavyland/sway "$@"
# -v $PWD/xdg:/var/lib/wavy
