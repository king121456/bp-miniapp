package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// WxAPI is the WeChat surface used by the backend; tests can replace it.
type WxAPI interface {
	Code2Session(ctx context.Context, code string) (string, error)
	SendSubscribe(ctx context.Context, openid, tpl, page string, data map[string]string) error
}

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
		slog.Error("wechat code2session transport error", "err", err)
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		slog.Error("wechat code2session decode error", "err", err)
		return "", err
	}
	if out.OpenID == "" {
		// Log full errcode/errmsg; never log AppSecret.
		slog.Error("wechat code2session failed",
			"errcode", out.ErrCode,
			"errmsg", out.ErrMsg,
			"http_status", resp.StatusCode,
			"appid", x.AppID,
		)
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
		slog.Error("wechat access_token transport error", "err", err)
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Token   string `json:"access_token"`
		Expires int    `json:"expires_in"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		slog.Error("wechat access_token decode error", "err", err)
		return "", err
	}
	if out.Token == "" {
		slog.Error("wechat access_token failed", "errcode", out.ErrCode, "errmsg", out.ErrMsg, "appid", x.AppID)
		return "", errors.New("no access token")
	}
	x.token, x.tokenExp = out.Token, time.Now().Add(time.Duration(out.Expires-300)*time.Second)
	return x.token, nil
}

func (x *Wx) SendSubscribe(ctx context.Context, openid, tpl, page string, data map[string]string) error {
	if tpl == "" {
		return errors.New("template id not configured")
	}
	tok, err := x.accessToken(ctx)
	if err != nil {
		slog.Error("wechat subscribe: no access token", "err", err)
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
		slog.Error("wechat subscribe transport error", "err", err, "tpl", tpl)
		return err
	}
	defer resp.Body.Close()
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		slog.Error("wechat subscribe decode error", "err", err)
		return err
	}
	if out.ErrCode != 0 {
		slog.Error("wechat subscribe send failed",
			"errcode", out.ErrCode,
			"errmsg", out.ErrMsg,
			"tpl", tpl,
			"openid_prefix", truncID(openid),
		)
		return fmt.Errorf("subscribe send %d %s", out.ErrCode, out.ErrMsg)
	}
	return nil
}

func truncID(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8] + "…"
}

// Fake is a test double; Code2Session returns "test_"+code.
type Fake struct{}

func (Fake) Code2Session(_ context.Context, code string) (string, error) {
	return "test_" + code, nil
}
func (Fake) SendSubscribe(context.Context, string, string, string, map[string]string) error {
	return nil
}
