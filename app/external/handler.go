package external

import (
	"dst-management-platform-api/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func modSearchGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type SearchForm struct {
		SearchType string `form:"searchType" json:"searchType"`
		SearchText string `form:"searchText" json:"searchText"`
		Page       int    `form:"page" json:"page"`
		PageSize   int    `form:"pageSize" json:"pageSize"`
	}
	var searchForm SearchForm
	if err := c.ShouldBindQuery(&searchForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if searchForm.SearchType == "id" {
		id, err := strconv.Atoi(searchForm.SearchText)
		if err != nil {
			logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}
		data, err := SearchModById(id, langStr)
		if err != nil {
			logger.Logger.Error("获取mod信息失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "search fail"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
		return
	}
	if searchForm.SearchType == "text" {
		data, err := SearchMod(searchForm.Page, searchForm.PageSize, searchForm.SearchText, langStr)
		if err != nil {
			logger.Logger.Error("获取mod信息失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "search fail"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
}
