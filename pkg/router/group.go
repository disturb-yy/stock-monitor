package router

import "github.com/gin-gonic/gin"

type Group struct {
	group *gin.RouterGroup
}

func NewGroup(group *gin.RouterGroup) *Group {
	return &Group{group: group}
}

func (g *Group) Use(handlers ...gin.HandlerFunc) *Group {
	g.group.Use(handlers...)
	return g
}

func (g *Group) UseIf(enabled bool, handlers ...gin.HandlerFunc) *Group {
	if enabled {
		g.Use(handlers...)
	}
	return g
}

func (g *Group) GET(relativePath string, handlers ...gin.HandlerFunc) *Group {
	g.group.GET(relativePath, handlers...)
	return g
}

// POST 注册 POST 路由并返回 Group 以支持链式调用。
func (g *Group) POST(relativePath string, handlers ...gin.HandlerFunc) *Group {
	g.group.POST(relativePath, handlers...)
	return g
}

func (g *Group) Native() *gin.RouterGroup {
	return g.group
}
