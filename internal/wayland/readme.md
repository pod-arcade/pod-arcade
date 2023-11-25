# Wayland Roots

This module interacts with [Wayland Roots](https://gitlab.freedesktop.org/wlroots/wlroots), a modular Wayland compositor library. It is the foundation for many Wayland compositors, such as [Sway](https://github.com/swaywm/sway). 

This module uses [go-wayland-scanner](https://github.com/rajveermalviya/go-wayland/cmd/go-wayland-scanner
) to generate native golang bindings for Wayland protocols. The generated bindings are then used to interact with Wayland Roots.