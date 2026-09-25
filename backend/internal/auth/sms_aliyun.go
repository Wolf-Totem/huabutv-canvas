package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"infinite-canvas/backend/internal/kernel"
)

func sendAliyunSMS(setting SMSSettingValue, phone, code string) error {
	phone, err := requireChinaMobile(phone)
	if err != nil {
		return err
	}
	params := map[string]string{
		"AccessKeyId":      setting.AccessKeyID,
		"Action":           "SendSms",
		"Format":           "JSON",
		"PhoneNumbers":     phone,
		"RegionId":         setting.Region,
		"SignName":         setting.SignName,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   kernel.NewID(),
		"SignatureVersion": "1.0",
		"TemplateCode":     setting.TemplateCode,
		"TemplateParam":    fmt.Sprintf(`{"code":"%s"}`, code),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Version":          "2017-05-25",
	}
	params["Signature"] = aliyunRPCSignature("POST", setting.AccessKeySecret, params)
	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Post("https://dysmsapi.aliyuncs.com/", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var payload struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("短信接口返回无法解析：%s", strings.TrimSpace(string(body)))
	}
	if !strings.EqualFold(payload.Code, "OK") {
		if payload.Message != "" {
			return fmt.Errorf("%s", payload.Message)
		}
		return fmt.Errorf("阿里云短信错误 %s", payload.Code)
	}
	return nil
}

func aliyunRPCSignature(method, accessKeySecret string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key == "Signature" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	query := make([]string, 0, len(keys))
	for _, key := range keys {
		query = append(query, aliyunPercentEncode(key)+"="+aliyunPercentEncode(params[key]))
	}
	canonical := strings.Join(query, "&")
	stringToSign := strings.ToUpper(method) + "&" + aliyunPercentEncode("/") + "&" + aliyunPercentEncode(canonical)
	mac := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func aliyunPercentEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
