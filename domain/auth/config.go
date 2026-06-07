package auth

import "time"

type Config struct {
	Enabled  bool
	Secret   string
	Issuer   string
	Audience string
	TokenTTL time.Duration
}
