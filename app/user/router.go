package user

import (
    "dst-management-platform-api/constants"
    "dst-management-platform-api/middleware"
    "github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
    v := r.Group(constants.ApiVersion)
    {
        auth := v.Group("user")
        {
            auth.POST("/register", h.registerPost)
            auth.POST("/login", h.loginPost)
            auth.GET("/menu", middleware.MWtoken(), h.menuGet)
        }
    }
}
