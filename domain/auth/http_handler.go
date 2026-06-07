package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/disturb-yy/stock-monitor/pkg/httputil"
	"github.com/disturb-yy/stock-monitor/pkg/logger"
	"github.com/gin-gonic/gin"
)

type GenerateTokenRequest struct {
	Subject string         `json:"subject"`
	Extra   map[string]any `json:"extra,omitempty"`
}

type GenerateTokenResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"tokenType"`
	ExpiresIn int    `json:"expiresIn"`
}

type HTTPHandler struct {
	service    *Service
	tokenTTL   int
}

func NewHTTPHandler(service *Service, tokenTTL int) *HTTPHandler {
	return &HTTPHandler{service: service, tokenTTL: tokenTTL}
}

func (h *HTTPHandler) GenerateToken(c *gin.Context) {
	var req GenerateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Response(c, http.StatusBadRequest, httputil.Resp{
			Code: httputil.ParamError,
			Msg:  "invalid request body",
		})
		return
	}

	req.Subject = strings.TrimSpace(req.Subject)
	if req.Subject == "" {
		httputil.Response(c, http.StatusBadRequest, httputil.Resp{
			Code: httputil.ParamError,
			Msg:  "subject is required",
		})
		return
	}

	token, err := h.service.GenerateToken(req.Subject, req.Extra)
	if err != nil {
		status := http.StatusInternalServerError
		code := httputil.InternalError
		msg := "generate token failed"
		if errors.Is(err, ErrReservedClaim) {
			status = http.StatusBadRequest
			code = httputil.ParamError
			msg = err.Error()
		}

		logger.Error(c.Request.Context(), "generate token failed", "error", err)
		httputil.Response(c, status, httputil.Resp{
			Code: code,
			Msg:  msg,
		})
		return
	}

	logger.Info(c.Request.Context(), "token generated", "subject", req.Subject)
	httputil.Response(c, http.StatusOK, httputil.Resp{
		Code: httputil.Success,
		Msg:  "success",
		Data: GenerateTokenResponse{
			Token:     token,
			TokenType: "Bearer",
			ExpiresIn: h.tokenTTL,
		},
	})
}
