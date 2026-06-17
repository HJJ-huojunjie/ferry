package process

import (
	"encoding/json"
	"ferry/global/orm"
	"ferry/models/process"
	"ferry/models/system"
	"ferry/pkg/logger"
	"ferry/pkg/notify"
	"fmt"
)

/*
  @Author : ferry-feishu-extension
  @Desc   : 转交工单通知（独立新增文件，不修改任何已有业务代码）
            当 InversionWorkOrder 完成工单转交后，调用本函数向「被转交人」发送
            飞书 / 钉钉 / 企微 / 邮件 等多通道通知，沿用流程绑定的通知类型 (process.notice)。
*/

// SendTransferNotify 发送工单转交通知（支持多人）
//   workOrderId : 工单ID
//   toUserIds   : 被转交人(新处理人)的 ferry 用户ID列表
//   operatorName: 转交操作者(当前登录用户)的昵称，用于通知正文
//   remarks     : 转交备注(可空)
//
// 该方法内部已加 recover，且全部以 goroutine 形式异步运行，
// 不会影响 InversionWorkOrder 主流程的事务/返回结果。
func SendTransferNotify(workOrderId int, toUserIds []int, operatorName string, remarks string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("转交工单通知发送 panic: %v", r)
			}
		}()

		// 1. 查询工单信息
		var workOrderInfo process.WorkOrderInfo
		err := orm.Eloquent.Model(&process.WorkOrderInfo{}).
			Where("id = ?", workOrderId).
			Find(&workOrderInfo).Error
		if err != nil {
			logger.Errorf("转交通知-查询工单信息失败: %v", err)
			return
		}

		// 2. 查询流程信息(取通知类型 notice 列表)
		var processInfo process.Info
		err = orm.Eloquent.Model(&process.Info{}).
			Where("id = ?", workOrderInfo.Process).
			Find(&processInfo).Error
		if err != nil {
			logger.Errorf("转交通知-查询流程信息失败: %v", err)
			return
		}

		var noticeList []int
		if len(processInfo.Notice) > 0 {
			if err = json.Unmarshal(processInfo.Notice, &noticeList); err != nil {
				logger.Errorf("转交通知-解析 notice 失败: %v", err)
				return
			}
		}
		if len(noticeList) == 0 {
			logger.Info("转交通知-流程未配置通知通道，跳过")
			return
		}

		// 3. 查询所有被转交人(收件人)
		var toUsers []system.SysUser
		err = orm.Eloquent.Model(&system.SysUser{}).
			Where("user_id in (?)", toUserIds).
			Find(&toUsers).Error
		if err != nil {
			logger.Errorf("转交通知-查询被转交人失败: %v", err)
			return
		}

		// 4. 查询工单创建人(用于通知正文中的「创建人」字段)
		var creatorUser system.SysUser
		err = orm.Eloquent.Model(&system.SysUser{}).
			Where("user_id = ?", workOrderInfo.Creator).
			Find(&creatorUser).Error
		if err != nil {
			logger.Errorf("转交通知-查询工单创建人失败: %v", err)
		}

		// 5. 组装通知体并发送给所有被转交人
		subject := fmt.Sprintf("您有一条被转交的待办工单，请及时处理")
		description := fmt.Sprintf("%s 已将工单转交给您处理，请及时跟进。", operatorName)
		if remarks != "" {
			description = fmt.Sprintf("%s 备注：%s", description, remarks)
		}

		bodyData := notify.BodyData{
			SendTo: map[string]interface{}{
				"userList": toUsers,
			},
			EmailCcTo:   []string{},
			Subject:     subject,
			Description: description,
			Classify:    noticeList,
			ProcessId:   workOrderInfo.Process,
			Id:          workOrderInfo.Id,
			Title:       workOrderInfo.Title,
			Creator:     creatorUser.NickName,
			Priority:    workOrderInfo.Priority,
			CreatedAt:   workOrderInfo.CreatedAt.Format("2006-01-02 15:04:05"),
			Remarks:     remarks,
		}

		if err = bodyData.SendNotify(); err != nil {
			logger.Errorf("转交工单通知发送失败: %v", err)
			return
		}
		logger.Infof("转交工单通知已发送: workOrderId=%d -> userIds=%v", workOrderId, toUserIds)
	}()
}
