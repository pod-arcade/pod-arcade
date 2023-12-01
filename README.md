# Pod-Arcade: Kubernetes-Native Retro Game Streaming

***IMPORTANT NOTE: Pod-Arcade is still in early development. It is not yet ready for general use. We make no guarantees about the stability of each new build, and much of the documentation may be missing or incomplete.***

Play Retro Games with your friends, directly in your browser!

## Overview

Pod-Arcade is an open-source project that enables you to stream games via RetroArch or other compatible software, running on Wayland, directly to your web browser.

It is designed to be deployed on Kubernetes, but can also be deployed using Docker or any other container platform.

There are two major components to Pod-Arcade:

* The Pod-Arcade Server — an MQTT server manages the game streaming sessions. Desktops and web browsers connect to this server in order to stream games.

* The Pod-Arcade Desktop — a desktop application that runs on Wayland and streams games to the Pod-Arcade Server.

## Getting Started

If you just want to get something up and running quicky, you have a few different options.

### Helm

We provide some reference helm charts for deploying Pod-Arcade on Kubernetes, at [pod-arcade/charts](https://github.com/pod-arcade/charts). This is what we use to deploy Pod-Arcade during development, and is likely the easiest way to get started.

### Docker

Run the server with this. You should be able to connect using https://localhost:8443. You may need to accept the self-signed certificate.
If that doesn't work, you may need to generate your own certificate and key, add that to your trust store, mount it into the container, and set the `TLS_CERT` and `TLS_KEY` environment variables to the path you mounted them to. Alternatively, [chrome has a flag that will allow you to ignore invalid certificates on localhost](chrome://flags/#allow-insecure-localhost).

```bash
docker run -it --rm --name pa-server \
  -p 1883:1883 \
  -p 8080:8080 \
  -p 8443:8443 \
  -e DESKTOP_PSK="theMagicStringUsedToAuthenticateDesktops" \
  -e CLIENT_PSK="thePasswordUsersPutInToConnect" \
  -e ICE_SERVERS='[{"urls":["stun:stun.l.google.com:19302"]}' \
  -e AUTH_REQUIRED="true" \
  -e SERVE_TLS="true" \
 ghcr.io/pod-arcade/server:main
```

and run an example retroarch client with:

```bash
docker volume create pa-desktop-dri
docker run -it --rm --user 0 --privileged --link pa-server:pa-server \
  -e WAYLAND_DISPLAY=wayland-1 \
  -e MQTT_HOST="ws://pa-server:8080/mqtt" \
  -e DESKTOP_ID=example-retroarch \
  -e DESKTOP_PSK="theMagicStringUsedToAuthenticateDesktops" \
  -e DISABLE_HW_ACCEL='false' \
  -e DISPLAY=':0' \
  -e DRI_DEVICE_MODE=MKNOD \
  -e FFMPEG_HARDWARE='1' \
  -e PGID='1000' \
  -e PUID='1000' \
  -e PULSE_SERVER='unix:/tmp/pulse/pulse-socket' \
  -e UINPUT_DEVICE_MODE=NONE \
  -e UNAME=ubuntu \
  -e WLR_BACKENDS=headless \
  -e WLR_NO_HARDWARE_CURSORS='1' \
  -e WLR_RENDERER=gles2 \
  -e XDG_RUNTIME_DIR=/tmp/sway \
  -v /dev/dri:/host/dev/dri \
  -v /dev/uinput:/host/dev/uinput \
  -v pa-desktop-dri:/dev/dri \
 ghcr.io/pod-arcade/desktop:main
```

### Docker Compose

There's docker-compose for running desktops in [pod-arcade/example-apps](https://github.com/pod-arcade/example-apps)

You'll need to set some of the environment variables to have it connect to the Pod-Arcade server. Just be careful which example applications you look at. Many of those pod-arcade/example-apps simply use the built in VNC server to stream the desktop to the browser, not Pod-Arcade. That's because it's much faster to do development that way, and will be compatible with pod-arcade if the VNC approach works.


## Screenshots

Heres some screenshots of Pod-Arcade in action:

### Homepage

<img src="assets/screenshots/homepage.png" width=720px />

### Gameplay

<img src="assets/screenshots/mario-kart-n64-1.png" width=720px />

<img src="assets/screenshots/mario-kart-n64-2.png" width=720px />

<img src="assets/screenshots/switch-screenshot-1.png" width=720px />

<img src="assets/screenshots/switch-screenshot-2.png" width=720px />

# License
Pod-Arcade is licensed under the MIT License - see the LICENSE file for details.
