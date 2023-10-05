# APIs
All APIs for interacting with the pod-arcade server, for initiating a WebRTC connection to a pod-arcade desktop, or for reporting session statistics is done through an MQTT connection. Once a WebRTC connection is established to a pod-arcade desktop, additional signaling and input events are sent over the WebRTC DataChannels.

- [MQTT](#mqtt)
  - [Authentication](#authentication)
  - [Server APIs](#server-apis)
    - [`server/ice-servers`](#serverice-servers)
  - [Desktop APIs](#desktop-apis)
    - [`desktops/{desktop-id}/status`](#desktopsdesktop-idstatus)
  - [Session APIs](#session-apis)
    - [`desktops/{desktop-id}/session/{session-id}/status`](#desktopsdesktop-idsessionsession-idstatus)
    - [`desktops/{desktop-id}/session/{session-id}/webrtc-offer`](#desktopsdesktop-idsessionsession-idwebrtc-offer)
    - [`desktops/{desktop-id}/session/{session-id}/webrtc-answer`](#desktopsdesktop-idsessionsession-idwebrtc-answer)
    - [`desktops/{desktop-id}/session/{session-id}/offer-ice-candidate` and `desktops/{desktop-id}/session/{session-id}/answer-ice-candidate`](#desktopsdesktop-idsessionsession-idoffer-ice-candidate-and-desktopsdesktop-idsessionsession-idanswer-ice-candidate)
    - [`desktops/{desktop-id}/session/{session-id}/stats/{stat}`](#desktopsdesktop-idsessionsession-idstatsstat)
- [WebRTC](#webrtc)
  - [DataChannel: `input`](#datachannel-input)
    - [Keyboard: `0x01`](#keyboard-0x01)
    - [Mouse: `0x02`](#mouse-0x02)
    - [Touchscreen: `0x03`](#touchscreen-0x03)
    - [Gamepad: `0x04`](#gamepad-0x04)
    - [Gamepad Rumble: `0x05`](#gamepad-rumble-0x05)

## MQTT
The MQTT server is running on the same server as the pod-arcade web server. The MQTT server is configured to use websockets, so you can connect to it from a browser.

### Authentication
All authentication to the MQTT api is done using MQTT username/password. For user connections, the client id must start with "user:", the username must be that of the user, and the password must be an access token bound to that same username. For desktop connections, the client id must start with "desktop:", the username must be the desktop id, and the password must be the desktop secret.

### Server APIs

#### `server/ice-servers`
Returns a list of ICE servers that can be used to establish a WebRTC connection. The payload is a JSON array of objects with the following properties:
- `urls`: A list of URLs that can be used to connect to the ICE server
- `username`: The username to use when authenticating with the ICE server
- `credential`: The password to use when authenticating with the ICE server

### Desktop APIs
A Desktop represents a single connectable instance of a virtual desktop. `{desktop-id}` represents a unique identifier for a specific desktop. This value is static for a desktop and can be any alphanumeric characters up to 32 in length.

#### `desktops/{desktop-id}/status`
Will be either "online" or "offline". When a desktop makes a connection to the MQTT server, it configures a [last will](https://www.hivemq.com/blog/mqtt-essentials-part-9-last-will-and-testament/) message as "offline" and then publishes an "online" message to this topic.

To get a list of desktops, simply subscribing to `desktops/+/status` will give you a list of all desktops, of which you can filter to just the ones that are online based on the returned status.

### Session APIs
A desktop may have zero or more sessions connected to it at a time. Sessions are identified by their `{session-id}`, which is a value that is randomly generated apon connection. This value is not static and will change each time a session connects. A session id can be any alphanumeric characters up to 32 in length.

#### `desktops/{desktop-id}/session/{session-id}/status`
Will be either "connecting", "online", or "offline". When a client makes a connection to the MQTT server, it configures a [last will](https://www.hivemq.com/blog/mqtt-essentials-part-9-last-will-and-testament/) message as "offline" and then publishes an "connecting" message to this topic. Once a WebRTC connection has been established (see below), an "online" message will be published.

To get a list of sessions, simply subscribing to `desktops/{desktop-id}/session/+/status` will give you a list of all sessions, of which you can filter to just the ones that are online based on the returned status.

#### `desktops/{desktop-id}/session/{session-id}/webrtc-offer`
Begins a WebRTC handshake with the pod-arcade desktop. The payload should be a UTF-8 encoded SDP offer obtained from `PeerConnection.localDescription.sdp`. This event may be triggered more than once to contain more ICE candidates as they are gathered.

Once sent, the desktop will respond with a `webrtc-answer` message containing a UTF-8 encoded SDP answer that can be passed into `PeerConnection.setRemoteDescription()`.

#### `desktops/{desktop-id}/session/{session-id}/webrtc-answer`
Response SDP from the pod-arcade desktop containing a UTF-8 encoded SDP answer that can be passed into `PeerConnection.setRemoteDescription()`. This event may be triggered more than once after more ICE candidates are gathered.

#### `desktops/{desktop-id}/session/{session-id}/offer-ice-candidate` and `desktops/{desktop-id}/session/{session-id}/answer-ice-candidate`
Both of these topics are used to send corresponding ice candidates to the other party. The payload should be a JSON encoded ICE candidate obtained from `RTCPeerConnection.onicecandidate`.

#### `desktops/{desktop-id}/session/{session-id}/stats/{stat}`
Reports a session statistic to the pod-arcade server. The `:stat` parameter can be any of the following values:

// TODO

## WebRTC
Once a WebRTC connection has been established, the following events will be made available over the WebRTC DataChannels.

### DataChannel: `input`
This channel is used to send input events to the pod-arcade desktop. The payload should be a byte structure with the first byte indicating the type of input, and the remaining bytes being the payload for that input type.

The DataChannel should be configured such that its messages are ordered (default). It should also be configured with protocol being `pod-arcade-input-v1`. It should be set to pre-negotiated, with the DataChannel's id being set to 0.

```js
var inputChannel = peerConnection.createDataChannel('input', {
  id: 0,
  negotiated: true,
  ordered: true,
  protocol: 'pod-arcade-input-v1',
});
```

#### Keyboard: `0x01`
Payload Format:
- Byte 0: `0x01`
- Byte 1: `0x00` for keydown, `0x01` for keyup
- Byte 2-3: Keycode (https://developer.mozilla.org/en-US/docs/Web/API/UI_Events/Keyboard_event_code_values)

#### Mouse: `0x02`
Payload Format:
- Byte 0: `0x02`
- Byte 1: Bitpacked button state
  - Bit 0: ButtonLeft
  - Bit 1: ButtonRight
  - Bit 2: ButtonMiddle
- Byte 2-5: X velocity (float32LE)
- Byte 6-9: Y velocity (float32LE)

#### Touchscreen: `0x03`
Payload Format:
- Byte 0: `0x03`
- Byte 1: `0x00` for touchstart, `0x01` for touchend
- Byte 2-3: X coordinate (uint16LE)
- Byte 4-5: Y coordinate (uint16LE)

#### Gamepad: `0x04`
Payload Format:
- Byte 0: `0x04`
- Byte 1: Gamepad index
- Byte 2-3: Bitpacked button state
  - Byte 2 Bit 0: ButtonNorth
  - Byte 2 Bit 1: ButtonSouth
  - Byte 2 Bit 2: ButtonWest
  - Byte 2 Bit 3: ButtonEast
  - Byte 2 Bit 4: ButtonBumperLeft
  - Byte 2 Bit 5: ButtonBumperRight
  - Byte 2 Bit 6: ButtonThumbLeft
  - Byte 2 Bit 7: ButtonThumbRight
  - Byte 3 Bit 0: ButtonSelect
  - Byte 3 Bit 1: ButtonStart
  - Byte 3 Bit 2: ButtonDpadUp
  - Byte 3 Bit 3: ButtonDpadDown
  - Byte 3 Bit 4: ButtonDpadLeft
  - Byte 3 Bit 5: ButtonDpadRight
  - Byte 3 Bit 6: ButtonMode
- Byte 4-7: Left Thumbstick X ([-1,1] float32LE)
- Byte 8-11: Left Thumbstick Y ([-1,1] float32LE)
- Byte 12-15: Right Thumbstick X ([-1,1] float32LE)
- Byte 16-19: Right Thumbstick Y ([-1,1] float32LE)
- Byte 20-23: Left Trigger ([-1,1] float32LE)
- Byte 24-27: Right Trigger ([-1,1] float32LE)

#### Gamepad Rumble: `0x05`
Even though this is in the "input" channel, it is actually more like an "output" sent from the desktop instead of from the client
Payload Format:
- Byte 0: `0x05`
- Byte 1: Gamepad index
- Byte 2-5: Intensity ([0,1], float32LE)
- Byte 6-9: Duration, in milliseconds (uint32LE)