package download

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteDownload(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	v1.Use(utils.MWDownloadToken())
	{
		download := v1.Group("download")
		download.Static("/backup", "./dmp_files/backup")
	}

	return r
}
