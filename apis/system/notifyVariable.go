package system

import (
	"ferry/models/system"
	"ferry/tools/app"
	"strconv"

	"github.com/gin-gonic/gin"
)

/*
  通知模板自定义变量管理接口
*/

// GetNotifyVariableList 获取所有自定义变量
func GetNotifyVariableList(c *gin.Context) {
	vars, err := system.GetNotifyVariableList()
	if err != nil {
		app.Error(c, -1, err, "获取通知变量列表失败")
		return
	}
	app.OK(c, vars, "")
}

// CreateNotifyVariable 创建变量
func CreateNotifyVariable(c *gin.Context) {
	var v system.NotifyVariable
	if err := c.ShouldBind(&v); err != nil {
		app.Error(c, -1, err, "参数绑定失败")
		return
	}
	if v.VarKey == "" {
		app.Error(c, -1, nil, "变量名不能为空")
		return
	}
	if err := system.CreateNotifyVariable(&v); err != nil {
		app.Error(c, -1, err, "创建变量失败（变量名是否重复?）")
		return
	}
	app.OK(c, v, "创建成功")
}

// UpdateNotifyVariable 更新变量
func UpdateNotifyVariable(c *gin.Context) {
	var v system.NotifyVariable
	if err := c.ShouldBind(&v); err != nil {
		app.Error(c, -1, err, "参数绑定失败")
		return
	}
	if err := system.UpdateNotifyVariable(&v); err != nil {
		app.Error(c, -1, err, "更新变量失败")
		return
	}
	app.OK(c, v, "更新成功")
}

// DeleteNotifyVariable 删除变量
func DeleteNotifyVariable(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.Error(c, -1, err, "参数错误")
		return
	}
	if err := system.DeleteNotifyVariable(id); err != nil {
		app.Error(c, -1, err, "删除变量失败")
		return
	}
	app.OK(c, nil, "删除成功")
}
