<template>
  <view class="pg">
    <view class="vb f12" v-if="fromVisit"><image src="/static/icons/calendar-g.svg" />复诊用：统计截止到今天（{{ to }}），可改选其他范围</view>
    <view class="card">
      <view class="seg">
        <view v-for="o in opts" :key="o.k" class="sg f14" :class="{ on: mode === o.k }" @tap="pickMode(o.k)">{{ o.t }}</view>
      </view>
      <view v-if="mode === 'custom'" class="cus">
        <picker mode="date" :value="from" :end="today" @change="e => setRange(e.detail.value, to)"><view class="pk f14 c0 num">{{ from }}</view></picker>
        <text class="f12 c9">至</text>
        <picker mode="date" :value="to" :end="today" @change="e => setRange(from, e.detail.value)"><view class="pk f14 c0 num">{{ to }}</view></picker>
      </view>
      <view class="cell nm"><text class="f14 c3">报告上的姓名</text><input class="ip f14 c0" v-model="name" placeholder="默认使用昵称，可修改" placeholder-class="phc" maxlength="12" @blur="saveName" @confirm="saveName" /></view>
      <view class="f10 c9">姓名只保存在本机，不会上传。</view>
    </view>
    <view class="wrap"><canvas canvas-id="rc" id="rc" class="cv" :style="{ width: W + 'px', height: H + 'px' }"></canvas></view>
    <view v-if="err" class="f12 c6 tc">{{ err }}</view>
    <view class="acts">
      <view class="btn ghost flex1" @tap="share">发给微信好友</view>
      <view class="btn flex1" @tap="save">保存到相册</view>
    </view>
    <view class="f10 c9 tc ft">报告由你自己的记录生成，仅供就诊时参考</view>
  </view>
</template>
<script>
import { guard, getReport, getReportName, setReportName, getRole, getProfile } from '../../api.js'
const p2 = n => String(n).padStart(2, '0')
const ymd = d => d.getFullYear() + '-' + p2(d.getMonth() + 1) + '-' + p2(d.getDate())
// 报告卡绘制：高度随内容变化（宽 360 设计坐标）
const CW = 360
const CC = { am: '#10C495', pm: '#7B6CF0', t1: '#333333', t2: '#666666', t3: '#999999', line: '#EEF1F0', soft: '#F5F8F7', brand: '#10C495' }
const iround = v => v == null ? null : Math.round(v)
function cardHeight(r) {
  const meds = Math.min((r.meds || []).length, 4)
  let h = 116 + 24 + 222 + 20 + 150 + 20 + 128 + 20 + (64 + Math.max(meds, 1) * 22) + 20
  if ((r.symptoms || []).length) h += 76
  return h + 54
}
function drawCard(ctx, r, name, k, img = {}) {
  const H = cardHeight(r), P = 20
  ctx.scale(k, k)
  const font = (size, bold) => { ctx.setFontSize(size); ctx.font = (bold ? 'bold ' : '') + size + 'px ' + (bold === 'num' ? '"DIN Alternate","Helvetica Neue",Arial,sans-serif' : 'sans-serif') }
  const txt = (s, x, y, size, color, align = 'left', bold) => { font(size, bold); ctx.setFillStyle(color); ctx.setTextAlign(align); ctx.fillText(String(s), x, y) }
  const tw = (s, size, bold) => { font(size, bold); return ctx.measureText(String(s)).width }
  const rr = (x, y, w, h, rad, color) => { ctx.setFillStyle(color); ctx.beginPath(); ctx.moveTo(x + rad, y); ctx.lineTo(x + w - rad, y); ctx.arc(x + w - rad, y + rad, rad, -Math.PI / 2, 0); ctx.lineTo(x + w, y + h - rad); ctx.arc(x + w - rad, y + h - rad, rad, 0, Math.PI / 2); ctx.lineTo(x + rad, y + h); ctx.arc(x + rad, y + h - rad, rad, Math.PI / 2, Math.PI); ctx.lineTo(x, y + rad); ctx.arc(x + rad, y + rad, rad, Math.PI, Math.PI * 1.5); ctx.closePath(); ctx.fill() }
  const hline = (x1, x2, y, color, w = 0.5, dash) => { ctx.setStrokeStyle(color); ctx.setLineWidth(w); ctx.setLineDash(dash || [], 0); ctx.beginPath(); ctx.moveTo(x1, y); ctx.lineTo(x2, y); ctx.stroke(); ctx.setLineDash([], 0) }
  const ymd = t => { const d = new Date(t), p = n => String(n).padStart(2, '0'); return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) }
  // 背景 + 头部
  ctx.setFillStyle('#FFFFFF'); ctx.fillRect(0, 0, CW, H)
  const g = ctx.createLinearGradient(0, 0, CW, 116); g.addColorStop(0, '#10C495'); g.addColorStop(0.5, '#34C9A3'); g.addColorStop(1, '#62E17D')
  ctx.setFillStyle(g); ctx.fillRect(0, 0, CW, 116)
  if (img.white) ctx.drawImage(img.white, P, 12, 15, 18)
  txt('压压记', P + (img.white ? 20 : 0), 26, 11, '#fff', 'left', true)
  txt('生成于 ' + ymd(r.generatedAt), CW - P, 26, 10, 'rgba(255,255,255,0.85)', 'right')
  txt('就诊报告卡', P, 56, 20, '#fff', 'left', true)
  txt('姓名  ' + (name || '—'), P, 80, 13, '#fff')
  txt(`${r.from} 至 ${r.to} · 共 ${r.days} 天`, P, 100, 12, 'rgba(255,255,255,0.9)')
  let y = 116 + 24
  const S = r.series || []
  const has = S.filter(d => d.morning || d.evening), few = has.length > 0 && has.length <= 7
  const Pt = few ? has : S
  // 图表：标题行 / 说明行 / 绘图区 / 日期行
  const chart = (y0, plotH, keys, title, unit, note) => {
    txt(title, P, y0 + 14, 15, CC.t1, 'left', true); txt(unit, P + tw(title, 15, true) + 6, y0 + 14, 10, CC.t3)
    // 图例（右上）
    const lg = (x, col, dash, label) => { ctx.setStrokeStyle(col); ctx.setLineWidth(2); ctx.setLineDash(dash, 0); ctx.beginPath(); ctx.moveTo(x, y0 + 10); ctx.lineTo(x + 16, y0 + 10); ctx.stroke(); ctx.setLineDash([], 0); ctx.setFillStyle(col); ctx.beginPath(); ctx.arc(x + 8, y0 + 10, 2.5, 0, 2 * Math.PI); ctx.fill(); txt(label, x + 21, y0 + 14, 10, CC.t2) }
    lg(CW - P - 86, CC.am, [], '早间'); lg(CW - P - 40, CC.pm, [4, 3], '晚间')
    if (note) txt(note, P, y0 + 32, 10, CC.t3)
    const top = y0 + (note ? 52 : 36), bot = top + plotH, L = P + 30, R = CW - P - 8
    const vals = []; Pt.forEach(d => ['morning', 'evening'].forEach(p => d[p] && keys.forEach(kk => d[p][kk] != null && vals.push(d[p][kk]))))
    let lo = vals.length ? Math.min(...vals) : 60, hi = vals.length ? Math.max(...vals) : 140
    lo = Math.floor((lo - 8) / 10) * 10; hi = Math.ceil((hi + 8) / 10) * 10; hi = lo + Math.max(30, Math.ceil((hi - lo) / 30) * 30) // 3 格，每格整十
    const Y = v => bot - (v - lo) / (hi - lo) * plotH
    const X = i => few ? L + (i + 0.5) * (R - L) / Pt.length : Pt.length > 1 ? L + i * (R - L) / (Pt.length - 1) : (L + R) / 2
    for (let i = 0; i <= 3; i++) { const v = lo + (hi - lo) * i / 3, yy = Y(v); hline(L, R, yy, CC.line, 0.5, i ? [3, 3] : []); txt(Math.round(v), L - 8, yy + 3.5, 9, CC.t3, 'right') }
    // 日期行（独立一行）
    const dy = bot + 16
    if (few) { let last = -1e9; Pt.forEach((d, i) => { const x = X(i), s = d.date.slice(5); if (x - last >= tw(s, 9) + 6) { txt(s, x, dy, 9, CC.t3, 'center'); last = x } }) }
    else if (S.length) { txt(S[0].date.slice(5), L, dy, 9, CC.t3, 'left'); txt(S[S.length - 1].date.slice(5), R, dy, 9, CC.t3, 'right') }
    if (!vals.length) txt('这段时间没有记录', (L + R) / 2, (top + bot) / 2 + 4, 11, CC.t3, 'center')
    for (const [p, col, dash] of [['morning', CC.am, []], ['evening', CC.pm, [4, 3]]]) for (const kk of keys) {
      ctx.setStrokeStyle(col); ctx.setLineWidth(1.6); ctx.setLineDash(dash, 0); let open = false; ctx.beginPath()
      Pt.forEach((d, i) => { const v = d[p] && d[p][kk]; if (v == null) return; open ? ctx.lineTo(X(i), Y(v)) : ctx.moveTo(X(i), Y(v)); open = true })
      ctx.stroke(); ctx.setLineDash([], 0)
      Pt.forEach((d, i) => {
        const v = d[p] && d[p][kk]; if (v == null) return
        const x = X(i), yy = Y(v), rad = few ? 4 : Pt.length > 40 ? 1.5 : 2.5
        ctx.setFillStyle('#fff'); ctx.beginPath(); ctx.arc(x, yy, rad + 1.5, 0, 2 * Math.PI); ctx.fill()
        ctx.setFillStyle(col); ctx.beginPath(); ctx.arc(x, yy, rad, 0, 2 * Math.PI); ctx.fill()
        if (few) { const below = kk === 'dia'; const ox = p === 'evening' && d.morning && d.morning[kk] != null ? 14 : 0; txt(Math.round(v), x + ox, below ? yy + 15 : yy - 8, 10, col, 'center', 'num') }
      })
    }
    return dy + 4
  }
  y = chart(y, 130, ['sys', 'dia'], '血压', 'mmHg', '每天上方的点为收缩压（高压），下方为舒张压（低压）')
  y += 20; hline(P, CW - P, y, CC.line); y += 20
  y = chart(y, 74, ['hr'], '心率', '次/分') ; y += 20
  // 平均值：三格
  txt('平均值', P, y + 14, 15, CC.t1, 'left', true); y += 28
  const cw = (CW - P * 2 - 16) / 3
  ;[['全部平均', r.average], ['早间（0–12点）', r.morning], ['晚间（12–24点）', r.evening]].forEach(([lab, a], i) => {
    const x = P + i * (cw + 8); rr(x, y, cw, 76, 10, CC.soft)
    txt(lab, x + 10, y + 18, 10, CC.t3)
    if (a) { txt(`${iround(a.sys)}/${iround(a.dia)}`, x + 10, y + 46, 20, i ? CC.t1 : CC.brand, 'left', 'num'); txt(a.hr != null ? `心率 ${iround(a.hr)}` : '心率 —', x + 10, y + 64, 10, CC.t2) }
    else txt('暂无记录', x + 10, y + 46, 12, CC.t3)
  })
  y += 76 + 18; txt(`共 ${r.count} 次测量 · 血压单位 mmHg · 心率单位 次/分`, P, y, 10, CC.t3); y += 26
  hline(P, CW - P, y - 6, CC.line); y += 14
  // 用药
  const ad = r.adherence || {}, ms = (r.meds || []).slice(0, 4)
  txt('用药与完成率', P, y + 14, 15, CC.t1, 'left', true)
  if (ad.planned) { const pc = iround(ad.rate) + '%'; txt(pc, CW - P, y + 18, 22, CC.brand, 'right', 'num'); txt(`已服 ${ad.taken} / 应服 ${ad.planned} 次${ad.makeup ? '，补记 ' + ad.makeup + ' 次' : ''}`, P, y + 36, 11, CC.t2) }
  else txt('未设置用药', P, y + 36, 11, CC.t3)
  y += 56
  ms.forEach(m => { y += 22; txt(`${m.name}${m.dose ? ' ' + m.dose : ''}`.slice(0, 16), P, y, 12, CC.t1); txt(`${(m.times || []).join(' ')}${m.rate != null ? '   ' + iround(m.rate) + '%' : ''}`, CW - P, y, 11, CC.t2, 'right') })
  if (!ms.length) y += 22
  if ((r.meds || []).length > 4) txt(`另有 ${r.meds.length - 4} 种未列出`, P, y + 16, 10, CC.t3)
  y += 28
  // 症状
  if ((r.symptoms || []).length) {
    hline(P, CW - P, y - 8, CC.line); y += 12
    txt('记录的症状', P, y + 14, 15, CC.t1, 'left', true); y += 30
    let x = P
    r.symptoms.slice(0, 6).forEach(s => { const t = `${s.name} ${s.count}次`, w = tw(t, 11) + 20; if (x + w > CW - P) return; rr(x, y, w, 24, 12, '#E8F8F2'); txt(t, x + w / 2, y + 16, 11, '#0C9A76', 'center'); x += w + 8 })
    y += 44
  }
  hline(P, CW - P, H - 52, CC.line)
  const bt = '压压记 · 血压记录工具', bw = tw(bt, 10, true) + (img.color ? 18 : 0), bx = CW / 2 - bw / 2
  if (img.color) ctx.drawImage(img.color, bx, H - 40, 13, 13)
  txt(bt, bx + (img.color ? 18 : 0), H - 30, 10, CC.brand, 'left', true)
  txt(r.footer || '数据由用户自行记录，仅供就诊参考', CW / 2, H - 12, 10, CC.t3, 'center')
  return H
}
export default {
  data() {
    const t = new Date(), f = new Date(t.getFullYear(), t.getMonth(), t.getDate() - 29)
    const W = Math.min(360, (uni.getSystemInfoSync().windowWidth || 375) - 32)
    return { opts: [{ k: 7, t: '近7天' }, { k: 30, t: '近30天' }, { k: 90, t: '近90天' }, { k: 'custom', t: '自定义' }], mode: 30, from: ymd(f), to: ymd(t), today: ymd(t), name: '', rep: null, err: '', fromVisit: false, W, H: Math.round(W * 16 / 9) }
  },
  onLoad(o) {
    if (!guard()) return; this.name = getReportName() || getProfile().nickname || ''
    if (getRole() === 'family') { uni.showToast({ title: '家人端不提供导出', icon: 'none' }); return setTimeout(() => uni.navigateBack(), 800) }
    // 从复诊提醒进入：结束日期为今天，默认往前 30 天（自定义范围，可改选）
    if (o && o.from === 'visit') { this.fromVisit = true; this.mode = 'custom' }
    this.load()
  },
  methods: {
    pickMode(k) { this.mode = k; this.load() },
    setRange(f, t) {
      const d = (new Date(t) - new Date(f)) / 864e5
      if (d < 0) return uni.showToast({ title: '开始日期需早于结束日期', icon: 'none' })
      if (d > 89) return uni.showToast({ title: '最长可选 90 天', icon: 'none' })
      this.from = f; this.to = t; this.load()
    },
    saveName() { setReportName(this.name.trim()); this.draw() },
    async load() {
      this.err = ''
      try { this.rep = await getReport(this.mode === 'custom' ? { from: this.from, to: this.to } : { days: this.mode }); this.draw() }
      catch (e) { this.err = (e && (e.message || e.errMsg)) || '生成失败，请稍后再试' }
    },
    draw() {
      const r = this.rep; if (!r) return
      const H = Math.round(this.W * cardHeight(r) / CW)
      const go = () => { const ctx = uni.createCanvasContext('rc', this); drawCard(ctx, r, this.name.trim(), this.W / CW, { white: '/static/brand/logo-white.png', color: '/static/brand/logo.png' }); ctx.draw() }
      if (H !== this.H) { this.H = H; this.$nextTick(() => setTimeout(go, 50)) } else go()
    },
    image() {
      return new Promise((ok, no) => uni.canvasToTempFilePath({ canvasId: 'rc', destWidth: 1080, destHeight: Math.round(1080 * this.H / this.W), success: r => ok(r.tempFilePath), fail: no }, this))
    },
    // #ifdef H5
    async download() {
      const p = await this.image(); const a = document.createElement('a'); a.href = p; a.download = `就诊报告卡_${this.rep.from}_${this.rep.to}.png`; document.body.appendChild(a); a.click(); a.remove()
      uni.showToast({ title: '图片已下载', icon: 'none' })
    },
    // #endif
    async save() {
      if (!this.rep) return
      // #ifdef H5
      return this.download()
      // #endif
      // #ifndef H5
      const p = await this.image()
      uni.saveImageToPhotosAlbum({ filePath: p, success: () => uni.showToast({ title: '已保存到相册', icon: 'none' }),
        fail: e => { if (/auth/.test(e.errMsg || '')) uni.showModal({ title: '需要相册权限', content: '请在设置中允许保存到相册', confirmColor: '#10C495', success: m => m.confirm && uni.openSetting() }) } })
      // #endif
    },
    async share() {
      if (!this.rep) return
      // #ifdef MP-WEIXIN
      const p = await this.image(); wx.showShareImageMenu({ path: p })
      // #endif
      // #ifdef H5
      uni.showToast({ title: '网页版不支持直接分享，已下载图片', icon: 'none' }); this.download()
      // #endif
    }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: linear-gradient(180deg, #DDF8EE 0, #F5F7F6 400rpx); padding: 24rpx 0 40rpx; }
.vb { display: flex; align-items: center; gap: 12rpx; margin: 0 32rpx 20rpx; padding: 16rpx 24rpx; border-radius: 20rpx; background: #E3F8F1; color: #0C9A76; } .vb image { width: 32rpx; height: 32rpx; flex: none; }
.seg { display: flex; background: #F2F4F3; border-radius: 24rpx; padding: 6rpx; }
.sg { flex: 1; text-align: center; height: 64rpx; line-height: 64rpx; border-radius: 20rpx; color: #666; }
.sg.on { background: #fff; color: #10C495; font-weight: 600; box-shadow: 0 4rpx 12rpx rgba(16,60,48,.08); }
.cus { display: flex; align-items: center; justify-content: center; gap: 20rpx; margin-top: 20rpx; }
.pk { background: #F5F7F6; border-radius: 16rpx; padding: 10rpx 28rpx; }
.num { font-family: "DIN Alternate", "Helvetica Neue", Arial, sans-serif; font-weight: 700; }
.nm { margin-top: 12rpx; } .ip { flex: 1; text-align: right; margin-left: 24rpx; }
.wrap { display: flex; justify-content: center; margin: 0 32rpx 24rpx; }
.cv { border-radius: 24rpx; overflow: hidden; box-shadow: 0 12rpx 40rpx rgba(16,60,48,.08); background: #fff; }
.acts { display: flex; gap: 24rpx; margin: 0 32rpx; } .flex1 { flex: 1; }
.tc { text-align: center; } .ft { margin-top: 24rpx; }
</style>
