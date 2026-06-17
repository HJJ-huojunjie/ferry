package system

import (
	"ferry/global/orm"
	"ferry/models/base"
	"ferry/pkg/logger"
)

/*
  通知模板自定义变量模型
  支持用户在管理界面动态添加变量，模板中通过 {{var_key}} 引用
  value_type:
    static - 静态文本，直接使用 value
    expr   - 表达式/含变量引用，支持嵌套 {{xxx}}（由渲染器递归替换）
*/

type NotifyVariable struct {
	base.Model
	VarKey      string `gorm:"column:var_key; type:varchar(64); not null; unique" json:"var_key" form:"var_key"`
	VarName     string `gorm:"column:var_name; type:varchar(128)" json:"var_name" form:"var_name"`
	ValueType   string `gorm:"column:value_type; type:varchar(16); default:'static'" json:"value_type" form:"value_type"`
	Value       string `gorm:"column:value; type:text" json:"value" form:"value"`
	Description string `gorm:"column:description; type:varchar(255)" json:"description" form:"description"`
	Enable      bool   `gorm:"column:enable; type:tinyint(1); default:1" json:"enable" form:"enable"`
}

func (NotifyVariable) TableName() string {
	return "sys_notify_variable"
}

// 获取所有通知变量
func GetNotifyVariableList() (vars []NotifyVariable, err error) {
	err = orm.Eloquent.Model(&NotifyVariable{}).Order("id ASC").Find(&vars).Error
	if err != nil {
		logger.Errorf("获取通知变量列表失败: %v", err)
	}
	return
}

// 获取所有启用的变量（供模板渲染调用）
func GetEnabledNotifyVariables() (vars []NotifyVariable, err error) {
	err = orm.Eloquent.Model(&NotifyVariable{}).
		Where("enable = ?", true).
		Find(&vars).Error
	return
}

// 创建变量
func CreateNotifyVariable(v *NotifyVariable) (err error) {
	if v.ValueType == "" {
		v.ValueType = "static"
	}
	err = orm.Eloquent.Create(v).Error
	return
}

// 更新变量
func UpdateNotifyVariable(v *NotifyVariable) (err error) {
	err = orm.Eloquent.Model(&NotifyVariable{}).
		Where("id = ?", v.Id).
		Updates(map[string]interface{}{
			"var_key":     v.VarKey,
			"var_name":    v.VarName,
			"value_type":  v.ValueType,
			"value":       v.Value,
			"description": v.Description,
			"enable":      v.Enable,
		}).Error
	return
}

// 删除变量
func DeleteNotifyVariable(id int) (err error) {
	err = orm.Eloquent.Where("id = ?", id).Delete(&NotifyVariable{}).Error
	return
}
