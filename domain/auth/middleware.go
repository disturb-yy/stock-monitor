package auth

import (
	"net/http"

	"github.com/disturb-yy/stock-monitor/pkg/httputil"
	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"

	ContextJWTSubject = "jwt_subject"
	ContextJWTClaims  = "jwt_claims"
	ContextPrincipal  = "auth_principal"
)

func JWTMiddleware(authenticator Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, err := authenticator.ValidateBearerToken(c.GetHeader(AuthorizationHeader))
		if err != nil {
			httputil.Response(c, http.StatusUnauthorized, httputil.Resp{
				Code: httputil.ParamError,
				Msg:  "unauthorized",
			})
			c.Abort()
			return
		}

		c.Set(ContextPrincipal, principal)
		c.Set(ContextJWTSubject, principal.Subject)
		c.Set(ContextJWTClaims, principal.Claims)
		c.Next()
	}
}
