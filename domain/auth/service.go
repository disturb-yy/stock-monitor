package auth

import "time"

type Authenticator interface {
	ValidateBearerToken(headerValue string) (*Principal, error)
}

type Service struct {
	config Config
}

func NewService(config Config) *Service {
	return &Service{config: config}
}

func (s *Service) GenerateToken(subject string, extra map[string]any) (string, error) {
	return s.GenerateTokenAt(subject, extra, time.Now())
}

func (s *Service) GenerateTokenAt(subject string, extra map[string]any, now time.Time) (string, error) {
	return GenerateJWTAt(subject, extra, s.config, now)
}

func (s *Service) ValidateToken(token string) (*Principal, error) {
	return s.ValidateTokenAt(token, time.Now())
}

func (s *Service) ValidateTokenAt(token string, now time.Time) (*Principal, error) {
	claims, err := ValidateJWTAt(token, JWTOptions{
		Secret:   s.config.Secret,
		Issuer:   s.config.Issuer,
		Audience: s.config.Audience,
	}, now)
	if err != nil {
		return nil, err
	}

	return NewPrincipal(claims), nil
}

func (s *Service) ValidateBearerToken(headerValue string) (*Principal, error) {
	return s.ValidateBearerTokenAt(headerValue, time.Now())
}

func (s *Service) ValidateBearerTokenAt(headerValue string, now time.Time) (*Principal, error) {
	claims, err := ValidateBearerTokenAt(headerValue, JWTOptions{
		Secret:   s.config.Secret,
		Issuer:   s.config.Issuer,
		Audience: s.config.Audience,
	}, now)
	if err != nil {
		return nil, err
	}

	return NewPrincipal(claims), nil
}
