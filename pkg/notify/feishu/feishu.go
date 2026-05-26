package feishu

import (
	"bytes"
	"encoding/json"
	"ferry/models/system"
	"ferry/pkg/logger"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

/*
  飞书通知实现
  - 自建应用单聊：通过 AppID + AppSecret 获取 tenant_access_token，然后发送消息给指定用户
  - 群机器人：通过 Webhook 地址直接发送群消息
*/

const (
	FeishuGetTokenURL    = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
	FeishuSendMsgURL     = "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=open_id"
	FeishuBatchGetIDURL  = "https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id?user_id_type=open_id"
	FeishuGetUserURL     = "https://open.feishu.cn/open-apis/contact/v3/users"
)

// ===== 飞书自建应用单聊 =====

type TenantTokenReq struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type TenantTokenResp struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
	Expire            int    `json:"expire"`
}

type FeishuMsgContent struct {
	Text string `json:"text"`
}

type FeishuSendMsgReq struct {
	ReceiveID string `json:"receive_id"`
	MsgType   string `json:"msg_type"`
	Content   string `json:"content"`
}

type FeishuSendMsgResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// GetTenantAccessToken 获取飞书 tenant_access_token
func GetTenantAccessToken(appID, appSecret string) (token string, err error) {
	reqBody := TenantTokenReq{
		AppID:     appID,
		AppSecret: appSecret,
	}
	b, _ := json.Marshal(reqBody)

	resp, err := http.Post(FeishuGetTokenURL, "application/json", bytes.NewBuffer(b))
	if err != nil {
		logger.Errorf("飞书获取token失败: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var tokenResp TenantTokenResp
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return
	}
	if tokenResp.Code != 0 {
		err = fmt.Errorf("飞书获取token失败: %s", tokenResp.Msg)
		return
	}
	token = tokenResp.TenantAccessToken
	return
}

// SendFeishuMsg 通过飞书自建应用发送单聊消息
// openIDs: 用户的飞书 OpenID 列表
func SendFeishuMsg(openIDs []string, title, content string) {
	// 从数据库获取飞书配置
	config, err := system.GetEnabledNotifyConfig(2) // 2 = 飞书单聊
	if err != nil {
		logger.Errorf("获取飞书通知配置失败: %v", err)
		return
	}

	token, err := GetTenantAccessToken(config.AppId, config.AppSecret)
	if err != nil {
		logger.Errorf("获取飞书token失败: %v", err)
		return
	}

	sendToOpenIDs(openIDs, title, content, token)
}

// SendFeishuMsgByPhone 通过手机号实时查询OpenID并发送飞书单聊消息（方案B：不依赖映射表）
func SendFeishuMsgByPhone(phones []string, title, content string) {
	if len(phones) == 0 {
		return
	}

	// 1. 获取飞书单聊配置
	config, err := system.GetEnabledNotifyConfig(2)
	if err != nil {
		logger.Errorf("获取飞书通知配置失败: %v", err)
		return
	}

	// 2. 实时根据手机号查询 OpenID
	result, err := BatchGetOpenIDByMobile(phones, config.AppId, config.AppSecret)
	if err != nil {
		logger.Errorf("实时查询飞书OpenID失败: %v", err)
		return
	}

	var openIDs []string
	for _, phone := range phones {
		if openID, ok := result[phone]; ok && openID != "" {
			openIDs = append(openIDs, openID)
		} else {
			logger.Warnf("手机号 %s 未在飞书中匹配到用户，跳过通知", phone)
		}
	}

	if len(openIDs) == 0 {
		logger.Warn("没有匹配到任何飞书用户，跳过发送")
		return
	}

	// 3. 获取token并发送消息
	token, err := GetTenantAccessToken(config.AppId, config.AppSecret)
	if err != nil {
		logger.Errorf("获取飞书token失败: %v", err)
		return
	}

	sendToOpenIDs(openIDs, title, content, token)
}

// sendToOpenIDs 内部函数，向指定 OpenID 列表发送消息
func sendToOpenIDs(openIDs []string, title, content, token string) {

	msgContent := FeishuMsgContent{
		Text: fmt.Sprintf("%s\n%s", title, content),
	}
	contentBytes, _ := json.Marshal(msgContent)

	for _, openID := range openIDs {
		if openID == "" {
			continue
		}
		reqBody := FeishuSendMsgReq{
			ReceiveID: openID,
			MsgType:   "text",
			Content:   string(contentBytes),
		}
		b, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", FeishuSendMsgURL, bytes.NewBuffer(b))
		if err != nil {
			logger.Errorf("飞书发送消息创建请求失败: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logger.Errorf("飞书发送消息失败: %v", err)
			continue
		}
		resp.Body.Close()
	}
	logger.Info("飞书单聊通知发送完成")
}

// ===== 飞书群机器人 =====

type FeishuWebhookMsg struct {
	MsgType string              `json:"msg_type"`
	Content FeishuWebhookContent `json:"content"`
}

type FeishuWebhookContent struct {
	Text string `json:"text"`
}

// SendFeishuGroupMsg 通过飞书群机器人 Webhook 发送群消息
func SendFeishuGroupMsg(title, content string) {
	// 从数据库获取飞书群配置
	config, err := system.GetEnabledNotifyConfig(3) // 3 = 飞书群
	if err != nil {
		logger.Errorf("获取飞书群通知配置失败: %v", err)
		return
	}

	webhookURL := config.AppId // Webhook 地址存储在 app_id 字段

	msg := FeishuWebhookMsg{
		MsgType: "text",
		Content: FeishuWebhookContent{
			Text: fmt.Sprintf("%s\n%s", title, content),
		},
	}
	b, _ := json.Marshal(msg)

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(b))
	if err != nil {
		logger.Errorf("飞书群消息发送失败: %v", err)
		return
	}
	defer resp.Body.Close()
	logger.Info("飞书群机器人通知发送完成")
}

// ===== 批量查询飞书OpenID =====

type BatchGetIDReq struct {
	Mobiles []string `json:"mobiles"`
}

// BatchGetIDResp 飞书批量查询ID返回
// 飞书返回格式: {"code":0,"data":{"user_list":[{"mobile":"138xxxx","user_id":"ou_xxx"}]}}
type BatchGetIDResp struct {
	Code int                  `json:"code"`
	Msg  string               `json:"msg"`
	Data BatchGetIDRespData   `json:"data"`
}

type BatchGetIDRespData struct {
	UserList []FeishuUserMatch `json:"user_list"`
}

type FeishuUserMatch struct {
	Mobile string `json:"mobile"`    // 请求时传的手机号
	UserID string `json:"user_id"`   // 飞书open_id
}

type FeishuUserResp struct {
	Code int                `json:"code"`
	Msg  string             `json:"msg"`
	Data FeishuUserData     `json:"data"`
}

type FeishuUserData struct {
	User FeishuUserInfo `json:"user"`
}

type FeishuUserInfo struct {
	OpenID string `json:"open_id"`
	UserID string `json:"user_id"`
	Mobile string `json:"mobile"`
	Name   string `json:"name"`
}

// GetUserOpenIDByMobile 通过手机号查询单个用户的飞书OpenID
func GetUserOpenIDByMobile(mobile, appID, appSecret string) (openID string, err error) {
	token, err := GetTenantAccessToken(appID, appSecret)
	if err != nil {
		return
	}

	url := fmt.Sprintf("%s/%s?user_id_type=open_id", FeishuGetUserURL, mobile)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("飞书查询用户失败: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var userResp FeishuUserResp
	err = json.Unmarshal(body, &userResp)
	if err != nil {
		return
	}
	if userResp.Code != 0 {
		err = fmt.Errorf("飞书查询用户失败: %s", userResp.Msg)
		return
	}
	openID = userResp.Data.User.OpenID
	return
}

// BatchGetOpenIDByMobile 批量通过手机号查询飞书OpenID
// mobiles: ferry用户手机号列表
// 返回: map[手机号]飞书OpenID
func BatchGetOpenIDByMobile(mobiles []string, appID, appSecret string) (result map[string]string, err error) {
	if len(mobiles) == 0 {
		return make(map[string]string), nil
	}

	token, err := GetTenantAccessToken(appID, appSecret)
	if err != nil {
		return
	}

	// 飞书通讯录中手机号带国际区号前缀，需要给纯11位手机号加上 +86 前缀
	// 构建双向映射：+86号码 <-> 原始号码
	prefixedToOriginal := make(map[string]string)
	var prefixedMobiles []string
	for _, m := range mobiles {
		var pm string
		if len(m) == 11 && !strings.HasPrefix(m, "+") {
			pm = "+86" + m
		} else {
			pm = m
		}
		prefixedMobiles = append(prefixedMobiles, pm)
		prefixedToOriginal[pm] = m
	}

	// 飞书API支持批量查询，每次最多50个
	result = make(map[string]string)
	batchSize := 50
	for i := 0; i < len(prefixedMobiles); i += batchSize {
		end := i + batchSize
		if end > len(prefixedMobiles) {
			end = len(prefixedMobiles)
		}
		batch := prefixedMobiles[i:end]

		reqBody := BatchGetIDReq{Mobiles: batch}
		b, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", FeishuBatchGetIDURL, bytes.NewBuffer(b))
		if err != nil {
			logger.Errorf("飞书批量查询创建请求失败: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logger.Errorf("飞书批量查询失败: %v", err)
			continue
		}

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		logger.Infof("[飞书批量查询] 请求手机号=%v, 返回原始响应: %s", batch, string(body))

		var batchResp BatchGetIDResp
		err = json.Unmarshal(body, &batchResp)
		if err != nil {
			logger.Errorf("飞书批量查询解析失败: %v", err)
			continue
		}
		if batchResp.Code != 0 {
			logger.Errorf("飞书批量查询失败: %s", batchResp.Msg)
			continue
		}

		for _, user := range batchResp.Data.UserList {
			if user.UserID != "" {
				// 将返回的手机号映射回原始格式（不带前缀）
				original, ok := prefixedToOriginal[user.Mobile]
				if ok {
					result[original] = user.UserID
				} else {
					// 飞书可能返回不带前缀的格式，直接用返回值
					result[user.Mobile] = user.UserID
				}
			}
		}
	}
	return
}
