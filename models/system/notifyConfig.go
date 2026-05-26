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
	Channel    int    `gorm:"column:channel; type:int(11); not null" json:"channel" form:"channel"`          // 渠道类型
	Name       string `gorm:"column:name; type:varchar(64)" json:"name" form:"name"`                         // 渠道名称
	Enable     bool   `gorm:"column:enable; type:tinyint(1); default:0" json:"enable" form:"enable"`         // 启用状态
	ConfigType string `gorm:"column:config_type; type:varchar(64)" json:"config_type" form:"config_type"`    // 配置类型
	AppId      string `gorm:"column:app_id; type:varchar(512)" json:"app_id" form:"app_id"`                  // 应用/机器人标识
	AppSecret  string `gorm:"column:app_secret; type:varchar(512)" json:"app_secret" form:"app_secret"`      // 密钥/签名/凭证
	Remark     string `gorm:"column:remark; type:varchar(512)" json:"remark" form:"remark"`                  // 备注
}

func (NotifyConfig) TableName() string {
	return "sys_notify_config"
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
	err = orm.Eloquent.Model(&NotifyConfig{}).
		Where("id = ?", config.Id).
		Updates(map[string]interface{}{
			"channel":     config.Channel,
			"name":        config.Name,
			"enable":      config.Enable,
			"config_type": config.ConfigType,
			"app_id":      config.AppId,
			"app_secret":  config.AppSecret,
			"remark":      config.Remark,
		}).Error
	return
}

// 删除通知配置
func DeleteNotifyConfig(id int) (err error) {
	err = orm.Eloquent.Where("id = ?", id).Delete(&NotifyConfig{}).Error
	return
}
