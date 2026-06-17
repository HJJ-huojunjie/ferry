package system

import (
	"ferry/global/orm"
	"ferry/models/system"
	"ferry/pkg/logger"
	"ferry/pkg/notify/feishu"
	"ferry/tools"
	"ferry/tools/app"

	"github.com/gin-gonic/gin"
)

/*
  @Author : lanyulei
*/

func GetInfo(c *gin.Context) {

	var roles = make([]string, 1)
	roles[0] = tools.GetRoleName(c)

	var permissions = make([]string, 1)
	permissions[0] = "*:*:*"

	var buttons = make([]string, 1)
	buttons[0] = "*:*:*"

	RoleMenu := system.RoleMenu{}
	RoleMenu.RoleId = tools.GetRoleId(c)

	var mp = make(map[string]interface{})
	mp["roles"] = roles
	if tools.GetRoleName(c) == "admin" || tools.GetRoleName(c) == "系统管理员" {
		mp["permissions"] = permissions
		mp["buttons"] = buttons
	} else {
		list, _ := RoleMenu.GetPermis()
		mp["permissions"] = list
		mp["buttons"] = list
	}

	sysuser := system.SysUser{}
	sysuser.UserId = tools.GetUserId(c)
	user, err := sysuser.Get()
	if err != nil {
		app.Error(c, -1, err, "")
		return
	}

	mp["introduction"] = " am a super administrator"

	mp["avatar"] = "https://wpimg.wallstcn.com/f778738c-e4f8-4870-b634-56703b4acafe.gif"
	if user.Avatar != "" {
		mp["avatar"] = user.Avatar
	}

	// 自动从飞书同步用户姓名：如果 nick_name 为空或等于 username，尝试通过手机号从飞书获取真实姓名
	displayName := user.NickName
	if (displayName == "" || displayName == user.Username) && user.Phone != "" {
		feishuConfig, cfgErr := system.GetEnabledNotifyConfig(2) // 2=飞书单聊
		if cfgErr == nil && feishuConfig.AppId != "" {
			feishuName, nameErr := feishu.GetFeishuUserNameByPhone(user.Phone, feishuConfig.AppId, feishuConfig.AppSecret)
			if nameErr == nil && feishuName != "" {
				displayName = feishuName
				// 缓存到数据库，避免每次都调用飞书API
				orm.Eloquent.Table("sys_user").Where("user_id = ?", user.UserId).
					Update("nick_name", feishuName)
				logger.Infof("已从飞书同步用户姓名: %s -> %s", user.Username, feishuName)
			}
		}
	}

	mp["userName"] = displayName
	mp["userId"] = user.UserId
	mp["deptId"] = user.DeptId
	mp["name"] = displayName

	app.OK(c, mp, "")
}
