package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type OpenIDConnectHook struct {
	OIDCServer   string
	OIDCClientId string
	ctx          context.Context

	oidcServerWellKnown map[string]interface{}
	oidcJWKSEndpoint    string

	jwkCache jwk.Cache

	mqtt.HookBase
}

// Called Automatically when calling NewOAuthHook
func (h *OpenIDConnectHook) SetupJWKS() error {
	if h.OIDCClientId == "" {
		return fmt.Errorf("no valid client id was provided")
	}

	host, err := url.Parse(h.OIDCServer)
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

	if err := json.Unmarshal(data, &h.oidcServerWellKnown); err != nil {
		return err
	}

	if h.oidcServerWellKnown["jwks_uri"] == nil {
		return fmt.Errorf("no jwks_uri was found on the oauth server")
	}
	h.oidcJWKSEndpoint = h.oidcServerWellKnown["jwks_uri"].(string)

	h.Log.Debug("Using JWKS_URI " + h.oidcJWKSEndpoint)

	if err := h.jwkCache.Register(h.oidcJWKSEndpoint); err != nil {
		return err
	}

	if _, err := h.jwkCache.Get(h.ctx, h.oidcJWKSEndpoint); err != nil {
		return err
	}

	return nil
}

func NewOIDCHook(ctx context.Context, oauthServer string, oauthClientId string) *OpenIDConnectHook {
	h := &OpenIDConnectHook{
		OIDCServer:   oauthServer,
		OIDCClientId: oauthClientId,
		ctx:          ctx,
		jwkCache:     *jwk.NewCache(ctx),
	}

	return h
}

func (h *OpenIDConnectHook) Init(config any) error {
	return h.SetupJWKS()
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

	keySet, err := h.jwkCache.Get(h.ctx, h.oidcJWKSEndpoint)
	if err != nil {
		h.Log.Warn("Failed to fetch JWK", "error", err)
		return false
	}

	t, err := jwt.ParseString(pass,
		jwt.WithKeySet(keySet),
		jwt.WithVerify(true),
		jwt.WithValidator(jwt.ValidatorFunc(func(ctx context.Context, t jwt.Token) jwt.ValidationError {
			if t.Subject() != user {
				return jwt.NewValidationError(fmt.Errorf("token sub %v did not match mqtt username %v", t.Subject(), user))
			}
			if !slices.Contains(t.Audience(), h.OIDCClientId) {
				return jwt.NewValidationError(fmt.Errorf("token aud %v does not contain %v", t.Audience(), h.OIDCClientId))
			}
			return nil
		})),
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
