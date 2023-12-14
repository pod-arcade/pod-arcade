package stun

import (
	"context"
	"fmt"
	"net"

	"github.com/pion/stun"
	"github.com/pod-arcade/pod-arcade/pkg/log"
)

var l = log.NewLogger("stun", nil)

// startSTUNServer starts a STUN server on the given port and handles shutdown via context.
func StartStunServer(ctx context.Context, port int) error {
	// Create a UDP listener
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	conn, err := net.ListenPacket("udp4", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	l.Info().Msgf("STUN server started on port %d.", port)

	// Handle incoming packets
	go func() {
		buffer := make([]byte, 1500)
		for {
			n, remoteAddr, err := conn.ReadFrom(buffer)
			if err != nil {
				select {
				case <-ctx.Done():
					// If context is cancelled, exit the loop
					return
				default:
					l.Warn().Msgf("Error reading from UDP: %s", err)
					continue
				}
			}

			// Handle the STUN request in a separate goroutine
			go func() {
				message := &stun.Message{Raw: append([]byte{}, buffer[:n]...)}
				if err := message.Decode(); err != nil {
					l.Error().Err(err).Msgf("Failed to decode message\n")
					return
				}

				transactionID := message.TransactionID

				// Process STUN message (this is a simplified example)
				if message.Type.Method != stun.MethodBinding || message.Type.Class != stun.ClassRequest {
					l.Warn().Msgf("Unsupported STUN message from %s\n", remoteAddr)
					return
				}

				remoteIP := remoteAddr.(*net.UDPAddr).IP
				remotePort := remoteAddr.(*net.UDPAddr).Port
				// Build a STUN response
				response, err := stun.Build(stun.NewTransactionIDSetter(transactionID), stun.BindingSuccess,
					stun.XORMappedAddress{IP: remoteIP, Port: remotePort},
				)
				if err != nil {
					l.Error().Err(err).Msgf("Failed to create STUN response: %s\n", err)
					return
				}
				l.Debug().Msgf("Sending STUN response to %s\n", remoteAddr)

				// Send the response
				if _, err := conn.WriteTo(response.Raw, remoteAddr); err != nil {
					l.Error().Err(err).Msgf("Failed to write response to %s: %s\n", remoteAddr, err)
				}
			}()
		}
	}()

	// Wait for context cancellation to shut down the server
	<-ctx.Done()
	fmt.Println("STUN server is shutting down.")
	return nil
}
