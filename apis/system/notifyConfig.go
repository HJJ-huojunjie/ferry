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
		&system.NotifyTemplate{},
	).Error
	if err != nil {
		logger.Errorf("初始化通知表失败: %v", err)
		app.Error(c, -1, err, "初始化通知表失败")
		return
	}
	app.OK(c, nil, "初始化成功")
}

// GetNotifyTemplates 获取所有通知模版
func GetNotifyTemplates(c *gin.Context) {
	channelType := c.Query("channel_type")
	var templates []system.NotifyTemplate
	var err error
	if channelType != "" {
		templates, err = system.GetNotifyTemplatesByChannel(channelType)
	} else {
		templates, err = system.GetNotifyTemplates()
	}
	if err != nil {
		app.Error(c, -1, err, "获取通知模版失败")
		return
	}
	app.OK(c, templates, "")
}

// CreateNotifyTemplate 创建通知模版
func CreateNotifyTemplate(c *gin.Context) {
	var tpl system.NotifyTemplate
	if err := c.ShouldBindJSON(&tpl); err != nil {
		app.Error(c, -1, err, "参数绑定失败")
		return
	}
	if err := system.CreateNotifyTemplate(&tpl); err != nil {
		logger.Errorf("创建模版失败: %v", err)
		app.Error(c, -1, err, "创建模版失败")
		return
	}
	app.OK(c, tpl, "创建成功")
}

// UpdateNotifyTemplate 更新通知模版
func UpdateNotifyTemplate(c *gin.Context) {
	var tpl system.NotifyTemplate
	if err := c.ShouldBindJSON(&tpl); err != nil {
		app.Error(c, -1, err, "参数绑定失败")
		return
	}
	if err := system.UpdateNotifyTemplate(&tpl); err != nil {
		logger.Errorf("更新模版失败: %v", err)
		app.Error(c, -1, err, "更新模版失败")
		return
	}
	app.OK(c, tpl, "更新成功")
}

// DeleteNotifyTemplate 删除通知模版
func DeleteNotifyTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.Error(c, -1, err, "参数错误")
		return
	}
	if err := system.DeleteNotifyTemplate(id); err != nil {
		app.Error(c, -1, err, "删除模版失败")
		return
	}
	app.OK(c, nil, "删除成功")
}

// SaveNotifyTemplates 批量保存通知模版（兼容旧接口）
func SaveNotifyTemplates(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		app.Error(c, -1, err, "参数绑定失败")
		return
	}
	for channelType, content := range req {
		// 查找该渠道是否有模版，有则更新第一个，没有则创建
		templates, _ := system.GetNotifyTemplatesByChannel(channelType)
		if len(templates) > 0 {
			templates[0].Content = content
			system.UpdateNotifyTemplate(&templates[0])
		} else {
			newTpl := system.NotifyTemplate{
				Name:        channelType + "默认模版",
				ChannelType: channelType,
				Content:     content,
			}
			system.CreateNotifyTemplate(&newTpl)
		}
	}
	app.OK(c, nil, "保存成功")
}

// GetNotifyTemplateVariables 获取通知模版可用变量列表（系统内置 + 用户自定义）
func GetNotifyTemplateVariables(c *gin.Context) {
	type TemplateVariable struct {
		Key     string `json:"key"`
		Desc    string `json:"desc"`
		Example string `json:"example"`
	}
	variables := []TemplateVariable{
		{Key: "{{title}}", Desc: "工单标题", Example: "服务器扩容申请"},
		{Key: "{{creator}}", Desc: "工单创建人", Example: "张三"},
		{Key: "{{priority}}", Desc: "工单优先级", Example: "紧急"},
		{Key: "{{created_at}}", Desc: "工单创建时间", Example: "2026-06-08 10:30:00"},
		{Key: "{{completed_at}}", Desc: "工单完成时间（仅完成模版可用）", Example: "2026-06-10 08:59:16"},
		{Key: "{{url}}", Desc: "工单处理链接", Example: "http://domain/#/process/handle-ticket?workOrderId=1&processId=1"},
		{Key: "{{subject}}", Desc: "通知主题", Example: "您有一条待办工单"},
		{Key: "{{description}}", Desc: "工单描述", Example: "请及时处理该工单"},
		{Key: "{{now}}", Desc: "当前时间（动态）", Example: "2026-06-10 17:00:00"},
		{Key: "{{today}}", Desc: "今天日期", Example: "2026-06-10"},
		{Key: "{{tomorrow}}", Desc: "明天日期", Example: "2026-06-11"},
		{Key: "{{字段model}}", Desc: "工单表单字段（使用字段的model key或中文标签名）", Example: "{{certName}} 或 {{证书名称}}"},
	}
	// 追加用户自定义变量
	customVars, _ := system.GetEnabledNotifyVariables()
	for _, v := range customVars {
		desc := v.VarName
		if v.Description != "" {
			if desc != "" {
				desc = desc + " - " + v.Description
			} else {
				desc = v.Description
			}
		}
		if desc == "" {
			desc = "自定义变量"
		}
		variables = append(variables, TemplateVariable{
			Key:     "{{" + v.VarKey + "}}",
			Desc:    "[自定义] " + desc,
			Example: v.Value,
		})
	}
	app.OK(c, variables, "")
}
