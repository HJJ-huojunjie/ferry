package system

import (
	"ferry/global/orm"
	"ferry/models/base"
	"ferry/pkg/logger"
)

/*
  第三方通知配置模型
  渠道类型:
    1 - 邮件 (系统内置)
    2 - 飞书 (自建应用单聊)
    3 - 飞书群 (群机器人)
    4 - 钉钉 (自建应用单聊)
    5 - 钉钉群 (群机器人)
    6 - 企微 (自建应用单聊)
    7 - 企微群 (群机器人)
*/

type NotifyConfig struct {
	base.Model
	Channel        int    `gorm:"column:channel; type:int(11); not null" json:"channel" form:"channel"`          // 渠道类型
	Name           string `gorm:"column:name; type:varchar(64)" json:"name" form:"name"`                         // 渠道名称
	Enable         bool   `gorm:"column:enable; type:tinyint(1); default:0" json:"enable" form:"enable"`         // 启用状态
	ConfigType     string `gorm:"column:config_type; type:varchar(64)" json:"config_type" form:"config_type"`    // 配置类型
	AppId          string `gorm:"column:app_id; type:varchar(512)" json:"app_id" form:"app_id"`                  // 应用/机器人标识
	AppSecret      string `gorm:"column:app_secret; type:varchar(512)" json:"app_secret" form:"app_secret"`      // 密钥/签名/凭证
	Remark         string `gorm:"column:remark; type:varchar(512)" json:"remark" form:"remark"`                  // 备注
	TemplateId     int    `gorm:"column:template_id; type:int(11); default:0" json:"template_id" form:"template_id"` // 待办通知模版ID
	DoneTemplateId int    `gorm:"column:done_template_id; type:int(11); default:0" json:"done_template_id" form:"done_template_id"` // 完成通知模版ID
}

func (NotifyConfig) TableName() string {
	return "sys_notify_config"
}

// 通知模版模型（支持多模版）
type NotifyTemplate struct {
	base.Model
	Name        string `gorm:"column:name; type:varchar(128)" json:"name" form:"name"`                           // 模版名称
	ChannelType string `gorm:"column:channel_type; type:varchar(32)" json:"channel_type" form:"channel_type"`    // 适用渠道: feishu, dingtalk, wecom, email
	Content     string `gorm:"column:content; type:text" json:"content" form:"content"`                          // 模版内容
}

func (NotifyTemplate) TableName() string {
	return "sys_notify_template"
}

// 获取所有通知配置
func GetNotifyConfigList() (configs []NotifyConfig, err error) {
	err = orm.Eloquent.Model(&NotifyConfig{}).Find(&configs).Error
	if err != nil {
		logger.Errorf("获取通知配置列表失败: %v", err)
	}
	return
}

// 根据渠道获取启用的通知配置
func GetEnabledNotifyConfig(channel int) (config NotifyConfig, err error) {
	err = orm.Eloquent.Model(&NotifyConfig{}).
		Where("channel = ? AND enable = ?", channel, true).
		First(&config).Error
	return
}

// 创建通知配置
func CreateNotifyConfig(config *NotifyConfig) (err error) {
	err = orm.Eloquent.Create(config).Error
	return
}

// 更新通知配置
func UpdateNotifyConfig(config *NotifyConfig) (err error) {
	updates := map[string]interface{}{
		"channel":          config.Channel,
		"name":             config.Name,
		"enable":           config.Enable,
		"config_type":      config.ConfigType,
		"app_id":           config.AppId,
		"remark":           config.Remark,
		"template_id":      config.TemplateId,
		"done_template_id": config.DoneTemplateId,
	}
	// app_secret 为空时不覆盖数据库原值（避免绑定/启停等静默调用清空密钥）
	if config.AppSecret != "" {
		updates["app_secret"] = config.AppSecret
	}
	err = orm.Eloquent.Model(&NotifyConfig{}).
		Where("id = ?", config.Id).
		Updates(updates).Error
	return
}

// 删除通知配置
func DeleteNotifyConfig(id int) (err error) {
	err = orm.Eloquent.Where("id = ?", id).Delete(&NotifyConfig{}).Error
	return
}

// ===== 通知模版 CRUD =====

// 获取所有通知模版
func GetNotifyTemplates() (templates []NotifyTemplate, err error) {
	err = orm.Eloquent.Model(&NotifyTemplate{}).Find(&templates).Error
	return
}

// 按渠道类型获取模版列表
func GetNotifyTemplatesByChannel(channelType string) (templates []NotifyTemplate, err error) {
	err = orm.Eloquent.Model(&NotifyTemplate{}).
		Where("channel_type = ?", channelType).
		Find(&templates).Error
	return
}

// 根据ID获取模版
func GetNotifyTemplateById(id int) (tpl NotifyTemplate, err error) {
	err = orm.Eloquent.Model(&NotifyTemplate{}).Where("id = ?", id).First(&tpl).Error
	return
}

// 创建通知模版
func CreateNotifyTemplate(tpl *NotifyTemplate) (err error) {
	err = orm.Eloquent.Create(tpl).Error
	return
}

// 更新通知模版
func UpdateNotifyTemplate(tpl *NotifyTemplate) (err error) {
	err = orm.Eloquent.Model(&NotifyTemplate{}).
		Where("id = ?", tpl.Id).
		Updates(map[string]interface{}{
			"name":         tpl.Name,
			"channel_type": tpl.ChannelType,
			"content":      tpl.Content,
		}).Error
	return
}

// 删除通知模版
func DeleteNotifyTemplate(id int) (err error) {
	err = orm.Eloquent.Where("id = ?", id).Delete(&NotifyTemplate{}).Error
	return
}
