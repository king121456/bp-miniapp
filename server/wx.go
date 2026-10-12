package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// WxAPI 是后端用到的两个微信接口，测试里可替换。
type WxAPI interface {
	Code2Session(ctx context.Context, code string) (string, error)
	SendSubscribe(ctx context.Context, openid, tpl, page string, data map[string]string) error
}

// Wx wraps the WeChat mini program server APIs.
type Wx struct {
	AppID, Secret string
	mu            sync.Mutex
	token         string
	tokenExp      time.Time
}

var httpc = &http.Client{Timeout: 8 * time.Second}

func (x *Wx) Code2Session(ctx context.Context, code string) (string, error) {
	q := url.Values{"appid": {x.AppID}, "secret": {x.Secret}, "js_code": {code}, "grant_type": {"authorization_code"}}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.weixin.qq.com/sns/jscode2session?"+q.Encode(), nil)
	resp, err := httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if out.OpenID == "" {
		return "", fmt.Errorf("code2session %d %s", out.ErrCode, out.ErrMsg)
	}
	return out.OpenID, nil
}

func (x *Wx) accessToken(ctx context.Context) (string, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.token != "" && time.Now().Before(x.tokenExp) {
		return x.token, nil
	}
	body, _ := json.Marshal(map[string]string{"grant_type": "client_credential", "appid": x.AppID, "secret": x.Secret})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.weixin.qq.com/cgi-bin/stable_token", bytes.NewReader(body))
	resp, err := httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Token   string `json:"access_token"`
		Expires int    `json:"expires_in"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Token == "" {
		return "", errors.New("no access token")
	}
	x.token, x.tokenExp = out.Token, time.Now().Add(time.Duration(out.Expires-300)*time.Second)
	return x.token, nil
}

// SendSubscribe sends one subscribe message. data keys must match the template's fields.
func (x *Wx) SendSubscribe(ctx context.Context, openid, tpl, page string, data map[string]string) error {
	if tpl == "" { // 模板 ID 没配就不推送
		return errors.New("template id not configured")
	}
	tok, err := x.accessToken(ctx)
	if err != nil {
		return err
	}
	d := map[string]any{}
	for k, v := range data {
		d[k] = map[string]string{"value": v}
	}
	body, _ := json.Marshal(map[string]any{"touser": openid, "template_id": tpl, "page": page, "data": d, "miniprogram_state": "formal", "lang": "zh_CN"})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token="+tok, bytes.NewReader(body))
	resp, err := httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if out.ErrCode != 0 {
		return fmt.Errorf("subscribe send %d %s", out.ErrCode, out.ErrMsg)
	}
	return nil
}
