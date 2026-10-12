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
      <view class="cell nm"><text class="f14 c3">报告上的姓名</text><input class="ip f14 c0" v-model="name" placeholder="选填，仅保存在本机" placeholder-class="phc" maxlength="12" @blur="saveName" @confirm="saveName" /></view>
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
import { guard, getReport, getReportName, setReportName, getRole } from '../../api.js'
const p2 = n => String(n).padStart(2, '0')
const ymd = d => d.getFullYear() + '-' + p2(d.getMonth() + 1) + '-' + p2(d.getDate())
const BW = 360, BH = 640 // 设计坐标（9:16）
const C = { am: '#37A5E6', pm: '#8A6FD8', ink: '#222', sub: '#888', line: '#EEF1F0', brand: '#10C495' }
export default {
  data() {
    const t = new Date(), f = new Date(t.getFullYear(), t.getMonth(), t.getDate() - 29)
    const W = Math.min(360, (uni.getSystemInfoSync().windowWidth || 375) - 32)
    return { opts: [{ k: 7, t: '近7天' }, { k: 30, t: '近30天' }, { k: 90, t: '近90天' }, { k: 'custom', t: '自定义' }], mode: 30, from: ymd(f), to: ymd(t), today: ymd(t), name: '', rep: null, err: '', fromVisit: false, W, H: Math.round(W * 16 / 9) }
  },
  onLoad(o) {
    if (!guard()) return; this.name = getReportName()
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
      const ctx = uni.createCanvasContext('rc', this); const k = this.W / BW
      ctx.scale(k, k)
      const txt = (s, x, y, size, color, align = 'left', bold) => { ctx.setFontSize(size); ctx.setFillStyle(color); ctx.setTextAlign(align); ctx.font = (bold ? 'bold ' : '') + size + 'px sans-serif'; ctx.fillText(String(s), x, y) }
      // 背景与头部
      ctx.setFillStyle('#FFFFFF'); ctx.fillRect(0, 0, BW, BH)
      const g = ctx.createLinearGradient(0, 0, BW, 96); g.addColorStop(0, '#10C495'); g.addColorStop(0.45, '#34C9A3'); g.addColorStop(1, '#62E17D')
      ctx.setFillStyle(g); ctx.fillRect(0, 0, BW, 96)
      txt('就诊报告卡', 20, 34, 20, '#fff', 'left', true)
      txt(this.name.trim() ? '姓名：' + this.name.trim() : '姓名：', 20, 58, 14, '#fff')
      txt(`记录时间 ${r.from} 至 ${r.to}（${r.days} 天）`, 20, 78, 12, '#fff')
      const gd = new Date(r.generatedAt); txt('生成于 ' + ymd(gd), BW - 20, 34, 10, '#fff', 'right')
      // 图例
      const legend = (y) => {
        ctx.setLineWidth(2); ctx.setStrokeStyle(C.am); ctx.setLineDash([], 0); ctx.beginPath(); ctx.moveTo(BW - 150, y - 4); ctx.lineTo(BW - 130, y - 4); ctx.stroke(); txt('早', BW - 126, y, 10, C.sub)
        ctx.setStrokeStyle(C.pm); ctx.setLineDash([4, 3], 0); ctx.beginPath(); ctx.moveTo(BW - 100, y - 4); ctx.lineTo(BW - 80, y - 4); ctx.stroke(); ctx.setLineDash([], 0); txt('晚', BW - 76, y, 10, C.sub)
      }
      const S = r.series || []
      const chart = (y0, h, keys, title, unit) => {
        txt(title, 20, y0 + 12, 14, C.ink, 'left', true); txt(unit, 20 + (title.length * 14) + 4, y0 + 12, 10, C.sub); legend(y0 + 12)
        const top = y0 + 22, bot = y0 + h, L = 44, R = BW - 20
        const vals = []; S.forEach(d => ['morning', 'evening'].forEach(p => d[p] && keys.forEach(kk => d[p][kk] != null && vals.push(d[p][kk]))))
        let lo = vals.length ? Math.min(...vals) : 60, hi = vals.length ? Math.max(...vals) : 140
        lo = Math.floor((lo - 5) / 10) * 10; hi = Math.ceil((hi + 5) / 10) * 10; if (hi - lo < 20) hi = lo + 20
        const Y = v => bot - (v - lo) / (hi - lo) * (bot - top), X = i => S.length > 1 ? L + i * (R - L) / (S.length - 1) : (L + R) / 2
        ctx.setLineWidth(0.5); ctx.setStrokeStyle(C.line); ctx.setLineDash([], 0)
        for (let i = 0; i <= 2; i++) { const v = lo + (hi - lo) * i / 2, y = Y(v); ctx.beginPath(); ctx.moveTo(L, y); ctx.lineTo(R, y); ctx.stroke(); txt(Math.round(v), L - 6, y + 3, 10, C.sub, 'right') }
        if (S.length) { txt(S[0].date.slice(5), L, bot + 12, 10, C.sub, 'left'); txt(S[S.length - 1].date.slice(5), R, bot + 12, 10, C.sub, 'right') }
        for (const [p, col, dash] of [['morning', C.am, []], ['evening', C.pm, [4, 3]]]) for (const kk of keys) {
          ctx.setStrokeStyle(col); ctx.setFillStyle(col); ctx.setLineWidth(1.5); ctx.setLineDash(dash, 0)
          let open = false; ctx.beginPath()
          S.forEach((d, i) => { const v = d[p] && d[p][kk]; if (v == null) { open = false; return } open ? ctx.lineTo(X(i), Y(v)) : ctx.moveTo(X(i), Y(v)); open = true })
          ctx.stroke(); ctx.setLineDash([], 0)
          S.forEach((d, i) => { const v = d[p] && d[p][kk]; if (v != null) { ctx.beginPath(); ctx.arc(X(i), Y(v), S.length > 40 ? 1 : 1.8, 0, 2 * Math.PI); ctx.fill() } })
        }
      }
      chart(108, 150, ['sys', 'dia'], '血压', 'mmHg'); txt('上方为收缩压，下方为舒张压', BW / 2, 270, 10, C.sub, 'center')
      chart(286, 80, ['hr'], '心率', '次/分')
      // 均值
      let y = 400; ctx.setFillStyle('#F5F7F6'); ctx.fillRect(16, y - 14, BW - 32, 64)
      const av = (a) => a ? `${a.sys}/${a.dia}` + (a.hr != null ? ` · ${a.hr}` : '') : '—'
      txt('全部平均', 28, y + 4, 10, C.sub); txt(av(r.average), 28, y + 24, 14, C.ink, 'left', true)
      txt('早间（0–12点）', 140, y + 4, 10, C.sub); txt(av(r.morning), 140, y + 24, 12, C.ink)
      txt('晚间', 250, y + 4, 10, C.sub); txt(av(r.evening), 250, y + 24, 12, C.ink)
      txt(`共 ${r.count} 次测量 · 血压 mmHg · 心率 次/分`, 28, y + 42, 10, C.sub)
      // 用药
      y = 472; txt('用药与完成率', 20, y, 14, C.ink, 'left', true)
      const ad = r.adherence || {}
      txt(ad.planned ? `已服 ${ad.taken} 次 / 应服 ${ad.planned} 次（${ad.rate}%），其中补记 ${ad.makeup || 0} 次` : '未设置用药', 20, y + 18, 11, '#444')
      const ms = (r.meds || []).slice(0, 3)
      ms.forEach((m, i) => { const yy = y + 36 + i * 16; txt(`${m.name}${m.dose ? ' ' + m.dose : ''} · ${(m.times || []).join(' ')}`.slice(0, 28), 20, yy, 11, '#666') })
      if ((r.meds || []).length > 3) txt(`另有 ${r.meds.length - 3} 种药品未列出`, 20, y + 36 + 3 * 16, 10, C.sub)
      // 症状（没有记录症状时整块不显示）
      if ((r.symptoms || []).length) {
        y = 576; txt('记录的症状', 20, y, 14, C.ink, 'left', true)
        txt(r.symptoms.map(s => `${s.name} ${s.count} 次`).join('  ·  ').slice(0, 30), 20, y + 18, 12, '#444')
      }
      // 页脚
      ctx.setStrokeStyle(C.line); ctx.setLineWidth(0.5); ctx.beginPath(); ctx.moveTo(20, BH - 26); ctx.lineTo(BW - 20, BH - 26); ctx.stroke()
      txt(r.footer || '数据由用户自行记录，仅供就诊参考', BW / 2, BH - 10, 10, C.sub, 'center')
      ctx.draw()
    },
    image() {
      return new Promise((ok, no) => uni.canvasToTempFilePath({ canvasId: 'rc', destWidth: 1080, destHeight: 1920, success: r => ok(r.tempFilePath), fail: no }, this))
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
