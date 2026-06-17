package notify

import (
	"bytes"
	"encoding/json"
	"ferry/global/orm"
	"ferry/models/process"
	"ferry/models/system"
	"ferry/pkg/logger"
	"ferry/pkg/notify/dingtalk"
	"ferry/pkg/notify/email"
	"ferry/pkg/notify/feishu"
	"ferry/pkg/notify/wecom"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/viper"
)

/*
  @Author : lanyulei
  @同时发送多种通知方式
*/

type BodyData struct {
	SendTo        interface{} // 接受人
	EmailCcTo     []string    // 抄送人邮箱列表
	Subject       string      // 标题
	Classify      []int       // 通知类型
	Id            int         // 工单ID
	Title         string      // 工单标题
	Creator       string      // 工单创建人
	Priority      int         // 工单优先级
	PriorityValue string      // 工单优先级
	CreatedAt     string      // 工单创建时间
	Content       string      // 通知的内容
	Description   string      // 表格上面的描述信息
	ProcessId     int         // 流程ID
	Domain        string      // 域名地址
	Remarks       string      // 处理/转交备注（由处理人填写）
	CompletedAt   string      // 工单完成时间（仅在到达结束节点发送完成通知时填充）
}

func (b *BodyData) ParsingTemplate() (err error) {
	// 读取模版数据
	var (
		buf bytes.Buffer
	)

	tmpl, err := template.ParseFiles("./static/template/email.html")
	if err != nil {
		return
	}

	b.Domain = viper.GetString("settings.domain.url")
	err = tmpl.Execute(&buf, b)
	if err != nil {
		return
	}

	b.Content = buf.String()

	return
}

// renderTemplateContent 根据渠道配置绑定的模版生成消息内容
// channelType: 渠道类型(1-7)
// 返回替换变量后的内容，如果没有绑定模版则返回空字符串
func (b *BodyData) renderTemplateContent(channelType int) string {
	// 查询该渠道的配置，获取绑定的 template_id
	config, err := system.GetEnabledNotifyConfig(channelType)
	if err != nil {
		return ""
	}
	if config.TemplateId == 0 {
		return ""
	}

	// 查询模版内容
	tpl, err := system.GetNotifyTemplateById(config.TemplateId)
	if err != nil || tpl.Content == "" {
		return ""
	}

	content := b.renderAllVars(tpl.Content)
	logger.Infof("模版变量替换完成，渠道=%d, 模版ID=%d", channelType, config.TemplateId)
	return content
}

// buildSystemVarMap 系统内置变量(BodyData 字段 + 衍生字段)
func (b *BodyData) buildSystemVarMap() map[string]string {
	url := fmt.Sprintf("%s/#/process/handle-ticket?workOrderId=%d&processId=%d", b.Domain, b.Id, b.ProcessId)
	remarks := b.Remarks
	if strings.TrimSpace(remarks) == "" {
		remarks = "无"
	}
	// 完成时间：工单未到结束节点时 b.CompletedAt 为空，此时显示“工单进行中”
	completedAt := b.CompletedAt
	if strings.TrimSpace(completedAt) == "" {
		completedAt = "工单进行中"
	}
	return map[string]string{
		"title":        b.Title,
		"creator":      b.Creator,
		"priority":     b.PriorityValue,
		"created_at":   b.CreatedAt,
		"completed_at": completedAt,
		"url":          url,
		"subject":      b.Subject,
		"description":  b.Description,
		"work_order_id": fmt.Sprintf("%d", b.Id),
		"process_id":   fmt.Sprintf("%d", b.ProcessId),
		"domain":       b.Domain,
		// 处理/转交备注（空值时显示“无”）
		"remarks":      remarks,
		"备注":         remarks,
		// 内置动态变量(无需在数据库中配置)
		"now":          time.Now().Format("2006-01-02 15:04:05"),
		"today":        time.Now().Format("2006-01-02"),
		"tomorrow":     time.Now().Add(24 * time.Hour).Format("2006-01-02"),
	}
}

// loadCustomVariables 从数据库加载用户自定义的变量
func (b *BodyData) loadCustomVariables() map[string]string {
	result := make(map[string]string)
	vars, err := system.GetEnabledNotifyVariables()
	if err != nil {
		logger.Errorf("加载自定义通知变量失败: %v", err)
		return result
	}
	for _, v := range vars {
		if v.VarKey == "" {
			continue
		}
		result[v.VarKey] = v.Value
		// 同时支持中文显示名作为别名(若设置)
		if v.VarName != "" && v.VarName != v.VarKey {
			result[v.VarName] = v.Value
		}
	}
	return result
}

// jsonEscapeValue 对字符串做 JSON 值安全转义（去除前后引号）
// 保证替换后的内容不会破坏 JSON 结构（如值中包含换行、引号、反斜杠等）
func jsonEscapeValue(s string) string {
	b, _ := json.Marshal(s)
	// json.Marshal 会输出 "escaped" 带引号，去掉首尾引号
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
}

// isJSONTemplate 检测模版内容是否为 JSON 格式（飞书卡片等）
func isJSONTemplate(content string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

// renderAllVars 统一执行变量替换：合并所有变量来源，嵌套替换最多 5 层
// 优先级：工单表单字段 > 自定义变量 > 系统内置变量
func (b *BodyData) renderAllVars(raw string) string {
	varMap := b.buildSystemVarMap()
	// 自定义变量覆盖系统变量
	for k, v := range b.loadCustomVariables() {
		varMap[k] = v
	}
	// 工单表单字段优先级最高(最贴近本工单数据)
	for k, v := range b.loadFormFieldValues() {
		varMap[k] = v
	}

	// 检测模版是否为 JSON 格式（飞书卡片等），若是则需要对替换值做 JSON 转义
	needJsonEscape := isJSONTemplate(raw)

	content := raw
	// 嵌套替换：自定义变量的 value 中可能引用其他 {{xxx}}，最多迭代 5 层防循环
	for i := 0; i < 5; i++ {
		before := content
		for name, value := range varMap {
			// 去除值末尾的换行/回车（textarea 保存时可能带入的意外尾部换行）
			replaceValue := strings.TrimRight(value, "\n\r")
			if needJsonEscape {
				replaceValue = jsonEscapeValue(replaceValue)
			}
			content = strings.ReplaceAll(content, "{{"+name+"}}", replaceValue)
			content = strings.ReplaceAll(content, "{{ "+name+" }}", replaceValue)
			content = strings.ReplaceAll(content, "@{{"+name+"}}", replaceValue)
			content = strings.ReplaceAll(content, "@{{ "+name+" }}", replaceValue)
		}
		if before == content {
			break
		}
	}

	// 最终清理：移除未能解析的占位符（避免原始 {{xxx}} 暴露给用户）
	// 仅在 JSON 模版中执行，防止破坏非 JSON 内容
	if needJsonEscape {
		for {
			idx := strings.Index(content, "{{")
			if idx == -1 {
				break
			}
			end := strings.Index(content[idx:], "}}")
			if end == -1 {
				break
			}
			placeholder := content[idx : idx+end+2]
			// 提取变量名用作显示
			varName := strings.TrimSpace(placeholder[2 : len(placeholder)-2])
			logger.Warnf("模版变量 {{%s}} 未找到对应值，已置空", varName)
			content = strings.Replace(content, placeholder, "", 1)
		}
	}

	return content
}

// loadFormFieldValues 从数据库加载工单的表单数据，返回变量名→字段值的映射
// 同时支持字段的 model key（如 certName）和 name label（如 证书名称）作为变量名
func (b *BodyData) loadFormFieldValues() map[string]string {
	result := make(map[string]string)
	if b.Id == 0 {
		return result
	}

	// 查询工单的所有表单数据
	var tplDataList []process.TplData
	err := orm.Eloquent.Model(&process.TplData{}).
		Where("work_order = ?", b.Id).
		Find(&tplDataList).Error
	if err != nil {
		logger.Errorf("加载工单表单数据失败: %v", err)
		return result
	}

	for _, tplData := range tplDataList {
		// 解析 form_structure 获取字段 model→name 映射
		var formStructure struct {
			List []struct {
				Model string `json:"model"`
				Name  string `json:"name"`
				Type  string `json:"type"`
			} `json:"list"`
		}
		if err := json.Unmarshal(tplData.FormStructure, &formStructure); err != nil {
			logger.Errorf("解析表单结构失败: %v", err)
			continue
		}

		// 解析 form_data 获取字段值
		var formData map[string]interface{}
		if err := json.Unmarshal(tplData.FormData, &formData); err != nil {
			logger.Errorf("解析表单数据失败: %v", err)
			continue
		}

		// 建立 model→name 映射
		modelToName := make(map[string]string)
		for _, field := range formStructure.List {
			if field.Name != "" && field.Model != "" {
				modelToName[field.Model] = field.Name
			}
		}

		// 遍历表单数据，同时用 model key 和字段标签名作为变量名
		for model, value := range formData {
			fieldName := modelToName[model]
			// 将值转为字符串
			var strValue string
			switch v := value.(type) {
			case string:
				strValue = v
			case float64:
				if v == float64(int64(v)) {
					strValue = fmt.Sprintf("%d", int64(v))
				} else {
					strValue = fmt.Sprintf("%v", v)
				}
			case nil:
				strValue = ""
			default:
				// 数组、对象等复杂类型转为JSON字符串
				jsonBytes, _ := json.Marshal(v)
				strValue = string(jsonBytes)
			}
			if strValue != "" {
				// 同时支持 model key 和 name label 作为模版变量名
				result[model] = strValue
				if fieldName != "" && fieldName != model {
					result[fieldName] = strValue
				}
			}
		}
	}

	logger.Infof("加载工单表单变量完成，工单ID=%d, 变量数=%d, 变量keys=%v", b.Id, len(result), func() []string {
		keys := make([]string, 0, len(result))
		for k := range result {
			keys = append(keys, k)
		}
		return keys
	}())
	return result
}

// defaultMsgContent 生成默认格式的消息内容
func (b *BodyData) defaultMsgContent() string {
	url := fmt.Sprintf("%s/#/process/handle-ticket?workOrderId=%d&processId=%d", b.Domain, b.Id, b.ProcessId)
	return fmt.Sprintf("工单标题：%s\n创建人：%s\n优先级：%s\n创建时间：%s\n处理链接：%s",
		b.Title, b.Creator, b.PriorityValue, b.CreatedAt, url)
}

// getMsgContent 获取消息内容：优先使用绑定模版，否则回退到默认格式
func (b *BodyData) getMsgContent(channelType int) string {
	content := b.renderTemplateContent(channelType)
	if content != "" {
		logger.Infof("渠道%d 使用绑定模版发送通知", channelType)
		return content
	}
	logger.Infof("渠道%d 未绑定模版，使用默认格式", channelType)
	return b.defaultMsgContent()
}

// getDoneMsgContent 获取工单完成通知的消息内容：优先使用 done_template_id 绑定的完成模版
func (b *BodyData) getDoneMsgContent(channelType int) string {
	content := b.renderDoneTemplateContent(channelType)
	if content != "" {
		logger.Infof("渠道%d 使用完成模版发送通知", channelType)
		return content
	}
	// 没有绑定完成模版，使用默认完成消息
	logger.Infof("渠道%d 未绑定完成模版，使用默认完成格式", channelType)
	url := fmt.Sprintf("%s/#/process/handle-ticket?workOrderId=%d&processId=%d", b.Domain, b.Id, b.ProcessId)
	return fmt.Sprintf("工单处理完成，请核对\n工单标题：%s\n创建人：%s\n优先级：%s\n创建时间：%s\n查看链接：%s",
		b.Title, b.Creator, b.PriorityValue, b.CreatedAt, url)
}

// renderDoneTemplateContent 渲染完成通知模版内容
func (b *BodyData) renderDoneTemplateContent(channelType int) string {
	config, err := system.GetEnabledNotifyConfig(channelType)
	if err != nil {
		return ""
	}
	if config.DoneTemplateId == 0 {
		return ""
	}

	tpl, err := system.GetNotifyTemplateById(config.DoneTemplateId)
	if err != nil || tpl.Content == "" {
		return ""
	}

	content := b.renderAllVars(tpl.Content)
	logger.Infof("完成模版变量替换完成，渠道=%d, 模版ID=%d", channelType, config.DoneTemplateId)
	return content
}

func (b *BodyData) SendNotify() (err error) {
	var (
		emailList []string
		phoneList []string
	)

	switch b.Priority {
	case 1:
		b.PriorityValue = "一般"
	case 2:
		b.PriorityValue = "紧急"
	case 3:
		b.PriorityValue = "非常紧急"
	}

	b.Domain = viper.GetString("settings.domain.url")

	for _, c := range b.Classify {
		switch c {
		case 1: // 邮件
			users := b.SendTo.(map[string]interface{})["userList"].([]system.SysUser)
			if len(users) > 0 {
				for _, user := range users {
					emailList = append(emailList, user.Email)
					phoneList = append(phoneList, user.Phone)
				}
				err = b.ParsingTemplate()
				if err != nil {
					logger.Errorf("模版内容解析失败，%v", err.Error())
					return
				}
				go email.SendMail(emailList, b.EmailCcTo, b.Subject, b.Content)
				dingtalkEnable := viper.GetBool("settings.dingtalk.enable")
				if dingtalkEnable {
					url := fmt.Sprintf("%s/#/process/handle-ticket?workOrderId=%d&processId=%d", b.Domain, b.Id, b.ProcessId)
					go dingtalk.SendDingMsg(phoneList, url, b.Title, b.Creator, b.PriorityValue, b.CreatedAt)
				}
			}
		case 2: // 飞书应用通知（实时根据手机号查询OpenID发送）
			users := b.SendTo.(map[string]interface{})["userList"].([]system.SysUser)
			if len(users) > 0 {
				var phones []string
				for _, user := range users {
					if user.Phone != "" {
						phones = append(phones, user.Phone)
					}
				}
				if len(phones) == 0 {
					logger.Warn("飞书应用通知：处理人未填写手机号，跳过")
					break
				}
				msgContent := b.getMsgContent(2)
				go feishu.SendFeishuMsgByPhone(phones, b.Subject, msgContent)
			}
		case 3: // 飞书群通知
			msgContent := b.getMsgContent(3)
			go feishu.SendFeishuGroupMsg(b.Subject, msgContent)
		case 4: // 钉钉应用通知
			users := b.SendTo.(map[string]interface{})["userList"].([]system.SysUser)
			if len(users) > 0 {
				for _, user := range users {
					phoneList = append(phoneList, user.Phone)
				}
				url := fmt.Sprintf("%s/#/process/handle-ticket?workOrderId=%d&processId=%d", b.Domain, b.Id, b.ProcessId)
				go dingtalk.SendDingMsg(phoneList, url, b.Title, b.Creator, b.PriorityValue, b.CreatedAt)
			}
		case 5: // 钉钉群通知
			msgContent := b.getMsgContent(5)
			go wecom.SendDingGroupMsg(b.Subject, msgContent)
		case 6: // 企微应用通知（暂未实现）
			logger.Warn("企微应用通知暂未实现实时查询模式，跳过")
		case 7: // 企微群通知
			msgContent := b.getMsgContent(7)
			go wecom.SendWecomGroupMsg(b.Subject, msgContent)
		}
	}
	return
}

// SendDoneNotify 发送工单完成通知（使用完成模版）
func (b *BodyData) SendDoneNotify() (err error) {
	var phoneList []string

	switch b.Priority {
	case 1:
		b.PriorityValue = "一般"
	case 2:
		b.PriorityValue = "紧急"
	case 3:
		b.PriorityValue = "非常紧急"
	}

	b.Domain = viper.GetString("settings.domain.url")

	for _, c := range b.Classify {
		switch c {
		case 1: // 邮件
			users := b.SendTo.(map[string]interface{})["userList"].([]system.SysUser)
			if len(users) > 0 {
				var emailList []string
				for _, user := range users {
					emailList = append(emailList, user.Email)
				}
				// 完成通知使用默认内容发送邮件
				go email.SendMail(emailList, nil, b.Subject, b.getDoneMsgContent(1))
			}
		case 2: // 飞书应用通知
			users := b.SendTo.(map[string]interface{})["userList"].([]system.SysUser)
			if len(users) > 0 {
				var phones []string
				for _, user := range users {
					if user.Phone != "" {
						phones = append(phones, user.Phone)
					}
				}
				if len(phones) == 0 {
					break
				}
				msgContent := b.getDoneMsgContent(2)
				go feishu.SendFeishuMsgByPhone(phones, b.Subject, msgContent)
			}
		case 3: // 飞书群通知
			msgContent := b.getDoneMsgContent(3)
			go feishu.SendFeishuGroupMsg(b.Subject, msgContent)
		case 4: // 钉钉应用通知
			users := b.SendTo.(map[string]interface{})["userList"].([]system.SysUser)
			if len(users) > 0 {
				for _, user := range users {
					phoneList = append(phoneList, user.Phone)
				}
				url := fmt.Sprintf("%s/#/process/handle-ticket?workOrderId=%d&processId=%d", b.Domain, b.Id, b.ProcessId)
				go dingtalk.SendDingMsg(phoneList, url, b.Title, b.Creator, b.PriorityValue, b.CreatedAt)
			}
		case 5: // 钉钉群通知
			msgContent := b.getDoneMsgContent(5)
			go wecom.SendDingGroupMsg(b.Subject, msgContent)
		case 7: // 企微群通知
			msgContent := b.getDoneMsgContent(7)
			go wecom.SendWecomGroupMsg(b.Subject, msgContent)
		}
	}
	return
}
