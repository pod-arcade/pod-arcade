# Userspace Devices (udev)

This package serves as a substitute for Linux's udev when you're operating a desktop environment within a container.

Note, most of the code for this package was modified and copied from: [pilebones' udev](https://github.com/pilebones/go-udev).

## Overview

In traditional host operating systems, udev is responsible for device management and permission control through a userspace process. However, containers usually do not run their own udev instances.

Another key challenge is that kernel events are global and not restricted to a specific namespace. When utilizing udev to emulate virtual peripherals like gamepads, mice, or keyboards, those devices become visible on the host system. Running individual udev instances for each container would result in each of these instances detecting events across all containers, thereby creating devices indiscriminately. This package aims to confine device creation to the specific container in which it operates.

Using a standalone udev instance within the container is not a foolproof solution either. Certain applications are designed to look for devices under specific names (e.g., /dev/input/js0-4). Employing udev to merely filter the devices established by the host system does not guarantee device name consistency. This is because another container might already have initialized the js0-4 devices, causing devices within our container to be assigned different names and, consequently, remain undetected.
