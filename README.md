# 血压记录小程序

uni-app（Vue3 + Vite）微信小程序前端，同一套代码可编译为 H5。

- 在线体验（H5 版，数据存在浏览器本地）：https://king121456.github.io/bp-miniapp/app/
- 设计预览与规范：https://king121456.github.io/bp-miniapp/

## 目录
- `miniapp/`：uni-app CLI 项目，源码在 `miniapp/src/`
  - `src/api.js` 数据层。`USE_LOCAL = true` 时全部存本地（`uni.setStorageSync`）；后端（Go + MongoDB）就绪后改为 `false` 并填写 `BASE`
  - `tools/icons.py` 生成线性图标 SVG 和 tabBar PNG
- `docs/`：GitHub Pages。`docs/app/` 为 H5 构建产物，`docs/mockups/` 为静态设计稿

## 运行
```bash
cd miniapp
npm install
npm run dev:mp-weixin   # 用微信开发者工具导入 dist/dev/mp-weixin
npm run dev:h5          # 浏览器预览
npm run build:mp-weixin # 产物 dist/build/mp-weixin
npm run build:h5        # 产物 dist/build/h5，复制到 docs/app/ 即可发布
```
也可以用 HBuilderX 打开 `miniapp/` 目录运行到微信小程序。发布前在 `src/manifest.json` 的 `mp-weixin.appid` 填入小程序 AppID。

## 说明
- 首次登录会生成约 10 天的示例数据，可在「我的 → 清除示例数据」中清空。
- 测量提醒在正式版需要用户同意微信订阅消息后才能推送。
- 本工具只记录数据，不对数值做任何判断。
