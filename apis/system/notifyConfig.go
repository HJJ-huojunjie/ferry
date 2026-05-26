package system

import (
	"ferry/global/orm"
	"ferry/models/system"
	"ferry/pkg/logger"
	"ferry/tools/app"
	"strconv"

	"github.com/gin-gonic/gin"
)

/*
  第三方通知配置管理接口
*/

// GetNotifyConfigList 获取通知配置列表
func GetNotifyConfigList(c *gin.Context) {
	configs, err := system.GetNotifyConfigList()
	if err != nil {
		app.Error(c, -1, err, "获取通知配置列表失败")
		return
	}
	app.OK(c, configs, "")
}

// CreateNotifyConfig 创建通知配置
func CreateNotifyConfig(c *gin.Context) {
	var config system.NotifyConfig
	if err := c.ShouldBind(&config); err != nil {
		app.Error(c, -1, err, "参数绑定失败")
		return
	}
	if err := system.CreateNotifyConfig(&config); err != nil {
		app.Error(c, -1, err, "创建通知配置失败")
		return
	}
	app.OK(c, config, "创建成功")
}

// UpdateNotifyConfig 更新通知配置
func UpdateNotifyConfig(c *gin.Context) {
	var config system.NotifyConfig
	if err := c.ShouldBind(&config); err != nil {
		app.Error(c, -1, err, "参数绑定失败")
		return
	}
	if err := system.UpdateNotifyConfig(&config); err != nil {
		app.Error(c, -1, err, "更新通知配置失败")
		return
	}
	app.OK(c, config, "更新成功")
}

// DeleteNotifyConfig 删除通知配置
func DeleteNotifyConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.Error(c, -1, err, "参数错误")
		return
	}
	if err := system.DeleteNotifyConfig(id); err != nil {
		app.Error(c, -1, err, "删除通知配置失败")
		return
	}
	app.OK(c, nil, "删除成功")
}

// InitNotifyTables 初始化通知相关数据库表
func InitNotifyTables(c *gin.Context) {
	err := orm.Eloquent.AutoMigrate(
		&system.NotifyConfig{},
	).Error
	if err != nil {
		logger.Errorf("初始化通知表失败: %v", err)
		app.Error(c, -1, err, "初始化通知表失败")
		return
	}
	app.OK(c, nil, "初始化成功")
}
