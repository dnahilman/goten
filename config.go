package goten

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	AppName        string
	BaseURL        string
	BasePath       string
	Secret         string
	Adapter        Adapter
	Plugins        []Plugin
	Session        SessionConfig
	Cookie         CookieConfig
	EmailPassword  EmailPasswordConfig
	TrustedOrigins []string
}

type SessionConfig struct {
	ExpiresIn time.Duration
	UpdateAge time.Duration
}

type CookieConfig struct {
	Name     string
	Domain   string
	Path     string
	Secure   *bool
	HTTPOnly *bool
	SameSite http.SameSite
}

type EmailPasswordConfig struct {
	Enabled           bool
	AutoSignIn        bool
	MinPasswordLength int
	MaxPasswordLength int
}

func (c *Config) validate() error {
	if c.BaseURL == "" {
		return errors.New("config: BaseURL required")
	}
	if c.Adapter == nil {
		return errors.New("config: Adapter required")
	}
	if len(c.Secret) < 32 {
		return errors.New("config: Secret must be at least 32 bytes")
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.AppName == "" {
		c.AppName = "Goten"
	}
	if c.BasePath == "" {
		c.BasePath = "/api/auth"
	}
	if c.Session.ExpiresIn == 0 {
		c.Session.ExpiresIn = 7 * 24 * time.Hour
	}
	if c.Session.UpdateAge == 0 {
		c.Session.UpdateAge = 24 * time.Hour
	}
	if c.Cookie.Name == "" {
		c.Cookie.Name = "goten_session"
	}
	if c.Cookie.Path == "" {
		c.Cookie.Path = "/"
	}
	if c.Cookie.SameSite == 0 {
		c.Cookie.SameSite = http.SameSiteLaxMode
	}
	if c.Cookie.Secure == nil {
		secure := strings.HasPrefix(c.BaseURL, "https://")
		c.Cookie.Secure = &secure
	}
	if c.Cookie.HTTPOnly == nil {
		t := true
		c.Cookie.HTTPOnly = &t
	}
	// EmailPassword defaults
	if !c.EmailPassword.Enabled {
		c.EmailPassword.Enabled = true
	}
	if !c.EmailPassword.AutoSignIn {
		c.EmailPassword.AutoSignIn = true
	}
	if c.EmailPassword.MinPasswordLength == 0 {
		c.EmailPassword.MinPasswordLength = 8
	}
	if c.EmailPassword.MaxPasswordLength == 0 {
		c.EmailPassword.MaxPasswordLength = 72
	}
}
