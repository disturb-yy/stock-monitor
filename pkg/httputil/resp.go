package httputil

import "github.com/gin-gonic/gin"

type Resp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

func Response(c *gin.Context, status int, resp Resp) {
	c.JSON(status, resp)
}
