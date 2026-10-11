# 血压小程序后端（Go + MongoDB）

运行：`cp .env.example .env` 填好后 `set -a; . ./.env; go run .`。Go 1.22+，MongoDB 4.4+。

统一返回 `{ "code": 0, "data": ... }`，出错为 `{ "code": <HTTP状态>, "message": "..." }`。
除登录外都需请求头 `Authorization: Bearer <token>`（30 天有效）。时间统一以 +08:00 的 RFC3339 返回，按北京时间计算"今天"。
备注、药名、剂量在库里用 AES-256-GCM 加密存储。

## 登录
- `POST /api/auth/login` `{ code, nickname?, consent: { agreedAt, version } }`（agreedAt 可传毫秒数字或 RFC3339 字符串） → `{ token, consent }`。
  首次登录必须带 consent，否则返回 403 `consent required`。
- `GET /api/me` 当前用户信息。

## 血压
- `GET /api/bp?from=&to=` 列表（倒序，最多 500 条）。每项 `{ id, time, sys, dia, hr, tags, note }`。
- `POST /api/bp` `{ sys, dia, hr, tags, note, time? }` → `{ record, needSubscribe }`。
- `PUT /api/bp/:id`，`DELETE /api/bp/:id`。
- `GET /api/bp/stats?range=week|month` → `{ average, morning, evening, daily[] }`，均值保留一位小数。上午为 0–12 点，其余为晚。只给数字，不含任何参考范围或评价。
- 校验只拦明显输错：高压 40–300，低压 20–200 且小于高压，心率 0 或 20–250。

## 用药
- `GET /api/meds` → 每项 `{ id, name, dose, times[], date, slots: [{ time, done, doneAt? }], done }`。
  `done` 指今天所有时间点都已打卡；单次是否打卡看 `slots`。
- `POST /api/meds` `{ name, dose, times: ["08:00","20:00"] }`，`PUT /api/meds/:id`，`DELETE /api/meds/:id`。
- `POST /api/meds/:id/checkin` `{ time: "08:00" }` → `{ needSubscribe }`。只有一个时间点时 time 可省略。重复打卡不报错。
- `GET /api/meds/checkins` 近 30 天打卡记录。

## 提醒与订阅消息
- `GET/PUT /api/reminders/measure` `{ enabled, morning: "07:30", evening: "20:00", morningEnabled, eveningEnabled }`。`enabled` 是总开关，早晚可单独关，不传视为开。
- `needSubscribe`：剩余推送次数少于 2 次时返回 `{ kind, templateId }`，否则为 `null`。前端拿 templateId 调 `wx.requestSubscribeMessage`。
- `POST /api/subscribe` `{ accepted: ["measure","med","family"] }` 用户点了允许后上报，每项次数 +1，返回各类剩余次数。
- `GET /api/todos` 没推送出去的提醒（次数用完或推送失败）；`POST /api/todos/:id/done` 关掉一条。录入血压、打卡后对应的待办会自动完成。
- 后台每分钟检查一次测量和用药时间点；每天 21:30 给绑定的家人发当天漏测、漏服汇总（用家人自己的授权次数）。

## 家人共享
- `POST /api/family/invite` 患者生成 6 位邀请码，24 小时有效，用一次即失效。
- `POST /api/family/bind` `{ code }` 家人绑定。
- `GET /api/family` → `{ members: [...], patients: [...] }`，分别是能看我的人、我能看的人。每项的 `id` 是绑定关系 id（解绑用）；`patients` 每项的 `patientId`（同 `userId`）才是下面查看接口里的 `:pid`。
- `DELETE /api/family/:id` 双方都能解绑。
- `GET /api/family/patients/:pid/bp`、`/bp/stats` 家人只读查看，未绑定返回 403。

## 就诊报告卡
- `GET /api/report?days=7|30|90`（默认 30）或 `?from=YYYY-MM-DD&to=YYYY-MM-DD`（最长 90 天）。
- 返回 `{ from, to, days, generatedAt, average, morning, evening, count, series: [{ date, morning, evening }], meds: [{ name, dose, times, planned, taken, rate }], adherence: { planned, taken, rate }, symptoms: [{ name, count }], footer }`。
- 完成率按药品当前的服药时间点乘以天数估算计划次数，rate 为百分比。
- 只限本人调用，家人没有对应接口。服务端只返回数字，图片由前端用 canvas 画，服务端不保存。姓名不经过后端。

## 账号
- `GET /api/account/export` 导出全部数据为 JSON。
- `DELETE /api/account` 注销，物理删除该用户所有记录、打卡、提醒和家人绑定。

## 订阅消息模板字段
默认字段名：测量 `thing1, time2`；用药 `thing1(药名), thing2(剂量), time3`；家人 `thing1, time2`。
在公众平台选好模板后，如果字段名不同，改 `scheduler.go` 里对应的 data 键。

## 部署
服务器上装好 Docker，`cp .env.example .env` 填好 AppID、Secret、密钥和模板 ID，然后 `docker compose up -d`。
服务只监听本机 8080，前面用 Nginx 配 HTTPS 证书反向代理到 `127.0.0.1:8080`，并把这个域名填到小程序后台的"request 合法域名"。
`DATA_KEY` 一旦用上就不能改，否则旧数据解不开，请单独备份。

## 本地 H5 联调（CORS）
仅当 `APP_ENV=dev` 且设置 `CORS_ORIGIN`（逗号分隔，或 `*`）时才返回 CORS 头并响应 OPTIONS 预检；生产环境不开启。
