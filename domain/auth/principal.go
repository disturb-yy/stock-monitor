package auth

type Principal struct {
	Subject string
	Claims  *JWTClaims
	Extra   map[string]any
}

func NewPrincipal(claims *JWTClaims) *Principal {
	if claims == nil {
		return &Principal{Extra: map[string]any{}}
	}

	return &Principal{
		Subject: claims.Subject,
		Claims:  claims,
		Extra:   claims.Extra,
	}
}
