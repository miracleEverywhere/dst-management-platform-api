package platform

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func (h *Handler) statusGet(c *gin.Context) {
	type Data struct {
		RunningTime float64 `json:"runningTime"`
		Memory      uint64  `json:"memory"`
	}

	t := time.Since(utils.StartTime).Seconds()
	mem := getRES()

	data := Data{
		RunningTime: t,
		Memory:      mem,
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}
