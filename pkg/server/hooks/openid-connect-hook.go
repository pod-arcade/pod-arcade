package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type OpenIDConnectHook struct {
	OAuthServer   string
	OAuthClientId string
	ctx           context.Context

	oauthServerWellKnown map[string]interface{}
	oauthJWKSEndpoint    string

	jwkCache jwk.Cache

	mqtt.HookBase
}

// Called Automatically when calling NewOAuthHook
func (h *OpenIDConnectHook) SetupJWKS() error {

	if h.OAuthClientId == "" {
		h.Log.Error("No Client ID was provided, you will be unable to verify connections")
	}

	host, err := url.Parse(h.OAuthServer)
	if err != nil {
		return err
	}
	host.Path = "/.well-known/openid-configuration"
	resp, err := http.Get(host.String())
	if err != nil {
		return err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &h.oauthServerWellKnown); err != nil {
		return err
	}

	if h.oauthServerWellKnown["jwks_uri"] == nil {
		return fmt.Errorf("no jwks_uri was found on the oauth server")
	}
	h.oauthJWKSEndpoint = h.oauthServerWellKnown["jwks_uri"].(string)

	h.Log.Debug("Using JWKS_URI " + h.oauthJWKSEndpoint)

	if err := h.jwkCache.Register(h.oauthJWKSEndpoint); err != nil {
		return err
	}

	if _, err := h.jwkCache.Get(h.ctx, h.oauthJWKSEndpoint); err != nil {
		return err
	}

	return nil
}

func NewOauthHook(ctx context.Context, oauthServer string, oauthClientId string) (*OpenIDConnectHook, error) {
	h := &OpenIDConnectHook{
		OAuthServer:   oauthServer,
		OAuthClientId: oauthClientId,
		ctx:           ctx,
		jwkCache:      *jwk.NewCache(ctx),
	}
	h.Log = slog.Default()

	if err := h.SetupJWKS(); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *OpenIDConnectHook) ID() string {
	return "OAuthHook"
}

func (h *OpenIDConnectHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnectAuthenticate,
		mqtt.OnACLCheck,
	}, []byte{b})
}

func (h *OpenIDConnectHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	user := string(pk.Connect.Username)
	pass := string(pk.Connect.Password)

	if !strings.HasPrefix(user, "user:") {
		h.Log.Debug("User %v does not match prefix `user:`", user)
		return false
	}

	user = strings.TrimPrefix(user, "user:")

	keySet, err := h.jwkCache.Get(h.ctx, h.oauthJWKSEndpoint)
	if err != nil {
		h.Log.Warn("Failed to fetch JWK", "error", err)
		return false
	}

	t, err := jwt.ParseString(pass,
		jwt.WithKeySet(keySet),
		jwt.WithVerify(true),
		jwt.WithValidator(jwt.ClaimValueIs("sub", user)),
		jwt.WithValidator(jwt.ClaimValueIs("aud", h.OAuthClientId)),
	)

	if err != nil {
		h.Log.Info("Failed to verify token", "error", err)
		return false
	}

	h.Log.Info("User logged in", "user", t.Subject())

	return true
}

func (h *OpenIDConnectHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	return true // do some checking later maybe
}
