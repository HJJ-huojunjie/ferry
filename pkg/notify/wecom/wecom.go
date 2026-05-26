package wecom

import (
	"bytes"
	"encoding/json"
	"ferry/models/system"
	"ferry/pkg/logger"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

/*
  企业微信通知实现
  - 自建应用单聊：通过 企业ID + AgentID + Secret 获取 access_token，然后发送应用消息
  - 群机器人：通过 Webhook 地址直接发送群消息
*/

const (
	WecomGetTokenURL = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	WecomSendMsgURL  = "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="
)

// ===== 企微自建应用单聊 =====

type WecomTokenResp struct {
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type WecomTextMsg struct {
	ToUser  string          `json:"touser"`
	MsgType string          `json:"msgtype"`
	AgentID int             `json:"agentid"`
	Text    WecomTextContent `json:"text"`
}

type WecomTextContent struct {
	Content string `json:"content"`
}

type WecomSendResp struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

// GetWecomAccessToken 获取企微 access_token
// corpID: 企业ID, secret: 应用Secret
func GetWecomAccessToken(corpID, secret string) (token string, err error) {
	url := fmt.Sprintf("%s?corpid=%s&corpsecret=%s", WecomGetTokenURL, corpID, secret)
	resp, err := http.Get(url)
	if err != nil {
		logger.Errorf("企微获取token失败: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var tokenResp WecomTokenResp
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return
	}
	if tokenResp.Errcode != 0 {
		err = fmt.Errorf("企微获取token失败: %s", tokenResp.Errmsg)
		return
	}
	token = tokenResp.AccessToken
	return
}

// SendWecomMsg 通过企微自建应用发送单聊消息
// userIDs: 企微 UserID 列表 ("|" 分隔)
func SendWecomMsg(wecomUserIDs []string, title, content string) {
	// 从数据库获取企微配置
	config, err := system.GetEnabledNotifyConfig(6) // 6 = 企微单聊
	if err != nil {
		logger.Errorf("获取企微通知配置失败: %v", err)
		return
	}

	// AppId 字段格式: "企业ID|AgentID"
	// AppSecret 字段: 应用Secret
	corpIDAndAgent := config.AppId
	secret := config.AppSecret

	// 解析企业ID和AgentID
	var corpID string
	var agentID int
	_, err = fmt.Sscanf(corpIDAndAgent, "%s", &corpID)
	// 尝试解析 "corpid|agentid" 格式
	n, _ := fmt.Sscanf(corpIDAndAgent, "%[^|]|%d", &corpID, &agentID)
	if n < 2 {
		logger.Errorf("企微配置格式错误，应为 '企业ID|AgentID' 格式: %s", corpIDAndAgent)
		return
	}

	token, err := GetWecomAccessToken(corpID, secret)
	if err != nil {
		logger.Errorf("获取企微token失败: %v", err)
		return
	}

	// 拼接用户ID
	toUser := ""
	for i, uid := range wecomUserIDs {
		if uid == "" {
			continue
		}
		if i > 0 {
			toUser += "|"
		}
		toUser += uid
	}
	if toUser == "" {
		return
	}

	msg := WecomTextMsg{
		ToUser:  toUser,
		MsgType: "text",
		AgentID: agentID,
		Text: WecomTextContent{
			Content: fmt.Sprintf("%s\n%s", title, content),
		},
	}
	b, _ := json.Marshal(msg)

	req, err := http.NewRequest("POST", WecomSendMsgURL+token, bytes.NewBuffer(b))
	if err != nil {
		logger.Errorf("企微发送消息创建请求失败: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("企微发送消息失败: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var sendResp WecomSendResp
	json.Unmarshal(body, &sendResp)
	if sendResp.Errcode != 0 {
		logger.Errorf("企微发送消息返回错误: %s", sendResp.Errmsg)
		return
	}
	logger.Info("企微单聊通知发送完成")
}

// ===== 企微群机器人 =====

type WecomWebhookMsg struct {
	MsgType string             `json:"msgtype"`
	Text    WecomWebhookContent `json:"text"`
}

type WecomWebhookContent struct {
	Content string `json:"content"`
}

// SendWecomGroupMsg 通过企微群机器人 Webhook 发送群消息
func SendWecomGroupMsg(title, content string) {
	// 从数据库获取企微群配置
	config, err := system.GetEnabledNotifyConfig(7) // 7 = 企微群
	if err != nil {
		logger.Errorf("获取企微群通知配置失败: %v", err)
		return
	}

	webhookURL := config.AppId // Webhook 地址存储在 app_id 字段

	msg := WecomWebhookMsg{
		MsgType: "text",
		Text: WecomWebhookContent{
			Content: fmt.Sprintf("%s\n%s", title, content),
		},
	}
	b, _ := json.Marshal(msg)

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(b))
	if err != nil {
		logger.Errorf("企微群消息发送失败: %v", err)
		return
	}
	defer resp.Body.Close()
	logger.Info("企微群机器人通知发送完成")
}

type DingWebhookMsg struct {
	MsgType  string             `json:"msgtype"`
	Markdown DingWebhookMarkdown `json:"markdown"`
}

type DingWebhookMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// SendDingGroupMsg 通过钉钉群机器人 Webhook 发送群消息
func SendDingGroupMsg(title, content string) {
	// 从数据库获取钉钉群配置
	config, err := system.GetEnabledNotifyConfig(5) // 5 = 钉钉群
	if err != nil {
		logger.Errorf("获取钉钉群通知配置失败: %v", err)
		return
	}

	webhookURL := config.AppId // Webhook 地址存储在 app_id 字段

	msg := DingWebhookMsg{
		MsgType: "markdown",
		Markdown: DingWebhookMarkdown{
			Title: title,
			Text:  fmt.Sprintf("### %s\n%s", title, content),
		},
	}
	b, _ := json.Marshal(msg)

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(b))
	if err != nil {
		logger.Errorf("钉钉群消息发送失败: %v", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		logger.Errorf("钉钉群机器人发送失败: %v", result["errmsg"])
		return
	}
	logger.Info("钉钉群机器人通知发送完成")
}
