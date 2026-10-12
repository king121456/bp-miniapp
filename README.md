# 血压记录小程序

uni-app（Vue3 + Vite）微信小程序前端 + Go/MongoDB 后端。同一套前端代码可编译为 H5。

- 在线体验（H5 版，本地模式，数据存在浏览器本地）：https://king121456.github.io/bp-miniapp/app/
- 设计预览与规范：https://king121456.github.io/bp-miniapp/

## 目录
- `miniapp/`：uni-app CLI 项目，源码在 `miniapp/src/`
  - `src/api.js` 数据层，本地模式与后端模式二选一（见下文）
  - `src/pages/report/` 就诊报告卡（canvas 生成 9:16 图片，姓名只存本机）
  - `tools/icons.py` 生成线性图标 SVG 和 tabBar PNG
- `server/`：Go + MongoDB 后端，接口说明见 [server/README.md](server/README.md)
- `docs/`：GitHub Pages。`docs/app/` 为 H5 构建产物（旧的本地演示模式构建，当前代码已无本地模式，需配好可访问的后端后重新构建），`docs/mockups/` 为静态设计稿

## 运行前端
```bash
cd miniapp
npm install
npm run dev:mp-weixin   # 用微信开发者工具导入 dist/dev/mp-weixin
npm run dev:h5          # 浏览器预览
npm run build:mp-weixin # 产物 dist/build/mp-weixin
npm run build:h5        # 产物 dist/build/h5（需要可访问的后端）
```
也可以用 HBuilderX 打开 `miniapp/` 目录运行到微信小程序。发布前在 `src/manifest.json` 的 `mp-weixin.appid` 填入小程序 AppID。

## 连接后端
前端所有数据都通过 Go 后端读写（没有本地/演示模式），本地只保存登录 token 等少量状态。后端地址由 `VITE_API_BASE` 指定，默认 `http://127.0.0.1:8080`：

```bash
# 临时
VITE_API_BASE=https://api.example.com npm run build:mp-weixin
# 或在 miniapp/ 下参照 .env.example 新建 .env.production.local（已被 .gitignore 忽略，不要提交）
VITE_API_BASE=https://api.example.com
```
- 小程序端用 `uni.login` 拿 code 调 `/api/auth/login`；H5 没有微信登录，会发送 `h5-dev`，只在后端 `APP_ENV=dev` 时可用。
- 订阅消息模板 ID 填在 `src/api.js` 的 `SUBSCRIBE_TMPL_IDS`（与后端 `WX_TPL_*` 一致）。
- 用 HBuilderX 时可以在运行配置里设置同样的环境变量。

## 本地运行后端
需要 Go 1.22+、MongoDB 4.4+。
```bash
cd server
cp .env.example .env      # 填写 JWT_SECRET、DATA_KEY（openssl rand -hex 32）等
set -a; . ./.env; set +a
go run .
B=http://localhost:8080 bash test.sh   # 接口自测
```
本机用浏览器联调 H5 时，设置 `APP_ENV=dev` 和 `CORS_ORIGIN=http://127.0.0.1:5173`（H5 开发服务器地址）。CORS 只在 dev 模式生效，小程序不需要。

## 用 Docker Compose 部署
```bash
cd server
cp .env.example .env      # 填 WX_APPID、WX_SECRET、JWT_SECRET、DATA_KEY、WX_TPL_*
docker compose up -d
```
服务监听本机 8080，用 Nginx 配 HTTPS 反向代理到 `127.0.0.1:8080`，并把域名填到小程序后台「request 合法域名」。

> ⚠️ **`DATA_KEY` 一旦用上就不能更换，丢了旧数据就解不开，请单独妥善备份。**
> ⚠️ **不要把 `.env`、密钥、编译出的二进制或数据目录提交到仓库**（`server/.gitignore` 已忽略这些）。

## 说明
- 本地模式首次登录会生成约 10 天的示例数据，可在「我的 → 清除示例数据」中清空。
- 测量提醒在正式版需要用户同意微信订阅消息后才能推送。
- 本工具只记录数据，不对数值做任何判断。
