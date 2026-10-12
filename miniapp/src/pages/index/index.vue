<template>
  <view class="pg">
    <view class="deco"></view>
    <view class="safe-top"></view>
    <view class="hd"><view class="f12 c6">{{ dateStr }}</view><view class="f20 c0 b">{{ greet }}，{{ name }}</view></view>


    <view class="vt" v-if="visit && visitTodo">
      <view class="vh"><view class="ic bg-b"><image src="/static/icons/calendar-b.svg" /></view><view class="flex1 vtx"><view class="f12 c9">复诊待办</view><view class="f20 c0 b">{{ visitLine }}</view></view></view>
      <view class="f14 c6 vsub" v-if="visitPlace">{{ visitPlace }}</view>
      <view class="btn vbtn" @tap="toCard">生成就诊卡片</view>
    </view>
    <view class="hero" v-if="latest" @tap="toRecord">
      <view class="t"><text>今日血压</text><view class="pill"><image src="/static/icons/check-w.svg" />{{ latestTime }} 已测量</view></view>
      <view class="bp"><text class="v num">{{ latest.sys }}</text><text class="s num">/</text><text class="v num">{{ latest.dia }}</text><text class="u">mmHg</text></view>
      <view class="lab"><text>收缩压</text><text>舒张压</text></view>
      <view class="hr">
        <view><view class="k">心率</view><view class="vv num">{{ latest.hr || '--' }}<text class="sm">次/分</text></view></view>
        <view><view class="k">今日测量</view><view class="vv num">{{ todayCount }}<text class="sm">次</text></view></view>
        <view><view class="k">本周测量</view><view class="vv num">{{ weekCount }}<text class="sm">次</text></view></view>
      </view>
    </view>
    <view class="hero emp" v-else>
      <view class="t"><text>今日血压</text><view class="pill">今天还没有记录</view></view>
      <view class="bp"><text class="v num o">--</text><text class="s num">/</text><text class="v num o">--</text><text class="u">mmHg</text></view>
      <view class="f12 et">测量后点下方按钮记录一次，这里会显示今天最新的一次</view>
      <view class="hr">
        <view><view class="k">今日测量</view><view class="vv num">0<text class="sm">次</text></view></view>
        <view><view class="k">本周测量</view><view class="vv num">{{ weekCount }}<text class="sm">次</text></view></view>
        <view></view>
      </view>
    </view>

    <view class="add" @tap="toEntry"><view class="ai"><image src="/static/icons/plus-w.svg" /></view>记录一次血压</view>

    <view class="sh"><text class="t">今日用药</text><view class="row f12 c9" @tap="toMeds">{{ meds.length ? doneCount + '/' + meds.length + ' 已服' : '去添加' }}<image class="arrow" src="/static/icons/right-9.svg" /></view></view>
    <view class="card mc">
      <view v-if="!meds.length" class="empty">还没有添加药品<view class="btn mini" style="margin:20rpx auto 0;width:200rpx" @tap="toMeds">添加药品</view></view>
      <view class="cell" v-for="m in meds" :key="m.id + m.time">
        <view class="ic bg-g"><image src="/static/icons/pill-g.svg" /></view>
        <view class="flex1 mn"><view class="f14 c0 b">{{ m.name }}</view><view class="f12 c9"><text class="num">{{ m.time }}</text> · {{ m.dose }}</view></view>
        <view class="ck done" v-if="m.done" @tap="m.makeup && (pick = m)"><image src="/static/icons/check-l.svg" />已服<text v-if="m.makeup" class="mk">补记</text></view>
        <view class="ck sk" v-else-if="m.status === 'skipped'" @tap="pick = m">未服</view>
        <view class="acts" v-else><text v-if="m.canSkip" class="skb" @tap="doSkip(m)">没吃</text>
          <view class="ck mkb" v-if="m.canMakeup" @tap="pick = m">补记</view>
          <view class="ck todo" v-else @tap="doCheck(m)">打卡</view></view>
      </view>
      <view class="miss" v-if="missCount" @tap="go('/pages/meds/makeup')"><text class="f14 c3 flex1">最近 7 天有 <text class="num b">{{ missCount }}</text> 次未打卡</text><text class="f12 cg">去补记</text><image class="arrow" src="/static/icons/right-g.svg" /></view>
    </view>

    <view class="card vc" v-if="visit && !visitTodo" @tap="go('/pages/visit/visit')">
      <view class="ic bg-b"><image src="/static/icons/calendar-b.svg" /></view>
      <view class="flex1 vtx"><view class="f16 c0 b">下次复诊：{{ md(visit.date) }}（{{ leftStr }}）</view><view class="f12 c9" v-if="visitPlace">{{ visitPlace }}</view></view>
      <image class="arrow" src="/static/icons/right-9.svg" />
    </view>

    <view class="mini">
      <view class="card" @tap="go('/pages/family/family')"><view class="ic bg-o"><image src="/static/icons/family-o.svg" /></view><view><view class="f14 c0 b">家人共享</view><view class="f10 c9">{{ famCount ? famCount + ' 位家人在关注' : '邀请家人查看' }}</view></view></view>
      <view class="card" @tap="go('/pages/remind/remind')"><view class="ic bg-b"><image src="/static/icons/bell-b.svg" /></view><view><view class="f14 c0 b">测量提醒</view><view class="f10 c9 num">{{ remindStr }}</view></view></view>
    </view>
    <view class="mini">
      <view class="card" @tap="go('/pages/visit/visit')"><view class="ic bg-b"><image src="/static/icons/calendar-b.svg" /></view><view><view class="f14 c0 b">复诊提醒</view><view class="f10 c9">{{ visit ? md(visit.date) + ' 复诊' : '记下复诊日期' }}</view></view></view>
      <view class="card" @tap="go('/pages/report/report')"><view class="ic bg-g"><image src="/static/icons/card-g.svg" /></view><view><view class="f14 c0 b">就诊报告卡</view><view class="f10 c9">看病时给医生看</view></view></view>
    </view>
    <slot-sheet :item="pick" @close="pick = null" @done="afterMakeup" />
    <view style="height:40rpx"></view>
  </view>
</template>
<script>
import { guard, listRecords, listMeds, checkin, listFamily, getRemind, getProfile, isToday, hhmm, weekStart, WEEK, makeupCount, skipSlot, getVisit, visitRemindAt, daysUntil, getRole } from '../../api.js'
export default {
  data: () => ({ records: [], meds: [], famCount: 0, remind: null, name: '', now: Date.now(), missCount: 0, visit: null, pick: null }),
  computed: {
    dateStr() { const d = new Date(this.now); return `${d.getMonth() + 1}月${d.getDate()}日 星期${WEEK[d.getDay()]}` },
    greet() { const h = new Date(this.now).getHours(); return h < 5 ? '夜深了' : h < 11 ? '早上好' : h < 13 ? '中午好' : h < 18 ? '下午好' : '晚上好' },
    today() { return this.records.filter(r => isToday(r.time)) },
    latest() { return this.today[0] }, latestTime() { return hhmm(this.latest.time) },
    todayCount() { return this.today.length }, weekCount() { const w = weekStart(); return this.records.filter(r => r.time >= w).length },
    // 复诊待办：从提醒时间（前一天 19:00）起到复诊当天结束显示（PRD 5.9）
    visitTodo() { return !!this.visit && this.now >= visitRemindAt(this.visit) },
    leftStr() { const n = daysUntil(this.visit.date); return n === 0 ? '今天' : n === 1 ? '明天' : '还有 ' + n + ' 天' },
    visitLine() { const n = daysUntil(this.visit.date); return (n === 0 ? '今天' : '明天') + '（' + this.md(this.visit.date) + '）复诊' },
    visitPlace() { const v = this.visit; return v ? [v.hospital, v.department].filter(Boolean).join(' ') : '' },
    doneCount() { return this.meds.filter(m => m.done).length },
    remindStr() { const r = this.remind; if (!r) return ''; const a = [r.morning.on && r.morning.time, r.evening.on && r.evening.time].filter(Boolean); return a.length ? a.join(' / ') : '未开启' }
  },
  onShow() {
    if (!guard()) return
    const role = getRole()
    if (!role) return uni.reLaunch({ url: '/pages/role/role' })
    if (role === 'family') return uni.reLaunch({ url: '/pages/family/home' })
    this.load()
  },
  methods: {
    async load() {
      this.now = Date.now(); this.name = getProfile().nickname
      ;[this.records, this.meds, this.remind] = await Promise.all([listRecords(), listMeds(), getRemind()])
      this.famCount = (await listFamily()).length
      this.missCount = await makeupCount()
      this.visit = await getVisit()
    },
    async doCheck(m) { await checkin(m.id, m.time); uni.showToast({ title: '已打卡', icon: 'success' }); this.meds = await listMeds() },
    async doSkip(m) {
      try { await skipSlot(m); uni.showToast({ title: '已记为未服', icon: 'none' }); this.afterMakeup() }
      catch (e) { uni.showToast({ title: (e && e.message) || '操作失败', icon: 'none' }) }
    },
    async afterMakeup() { this.pick = null; this.meds = await listMeds(); this.missCount = await makeupCount() },
    md(s) { const [, m, d] = s.split('-').map(Number); return `${m}月${d}日` },
    toCard() { uni.navigateTo({ url: '/pages/report/report?from=visit' }) },
    toEntry() { uni.navigateTo({ url: '/pages/entry/entry' }) },
    toMeds() { uni.navigateTo({ url: '/pages/meds/meds' }) },
    toRecord() { uni.switchTab({ url: '/pages/record/record' }) },
    go(url) { uni.navigateTo({ url }) }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: linear-gradient(180deg, #DDF8EE 0, #EEF9F4 360rpx, #F5F7F6 640rpx); position: relative; overflow: hidden; }
.deco { position: absolute; top: -120rpx; right: -160rpx; width: 520rpx; height: 520rpx; border-radius: 50%; background: radial-gradient(circle, rgba(167,255,185,.55), rgba(112,232,201,0) 70%); }
.hd { position: relative; padding: 16rpx 40rpx 32rpx; }
.hero { position: relative; margin: 0 32rpx 24rpx; border-radius: 40rpx; padding: 36rpx 36rpx 32rpx; color: #fff; overflow: hidden; background: linear-gradient(120deg, #10C495 0%, #34C9A3 45%, #62E17D 100%); box-shadow: 0 24rpx 56rpx -16rpx rgba(16,196,149,.55); }
.hero::before { content: ""; position: absolute; right: -60rpx; bottom: -120rpx; width: 360rpx; height: 360rpx; border-radius: 50%; border: 52rpx solid rgba(255,255,255,.09); }
.t { display: flex; justify-content: space-between; align-items: center; font-size: 24rpx; opacity: .92; }
.pill { display: inline-flex; align-items: center; gap: 8rpx; background: rgba(255,255,255,.2); padding: 6rpx 16rpx; border-radius: 20rpx; font-size: 20rpx; }
.pill image { width: 20rpx; height: 20rpx; }
.bp { display: flex; align-items: flex-end; margin: 20rpx 0 4rpx; }
.bp .v { font-size: 88rpx; line-height: 1; } .bp .v.o { opacity: .7; }
.bp .s { font-size: 56rpx; line-height: 1.1; opacity: .6; margin: 0 12rpx; }
.bp .u { font-size: 24rpx; opacity: .85; margin: 0 0 10rpx 12rpx; }
.lab { display: flex; gap: 88rpx; font-size: 20rpx; opacity: .85; margin-bottom: 28rpx; }
.et { opacity: .9; margin: 8rpx 0 28rpx; }
.hr { position: relative; display: flex; border-top: 1rpx solid rgba(255,255,255,.3); padding-top: 24rpx; }
.hr > view { flex: 1; } .k { font-size: 20rpx; opacity: .85; } .vv { font-size: 40rpx; line-height: 1.2; }
.sm { font-size: 20rpx; font-weight: 400; opacity: .85; margin-left: 4rpx; font-family: "PingFang SC", sans-serif; }
.add { margin: 0 32rpx 40rpx; height: 96rpx; border-radius: 48rpx; background: #fff; display: flex; align-items: center; justify-content: center; gap: 12rpx; color: #10C495; font-size: 32rpx; font-weight: 600; box-shadow: 0 2rpx 4rpx rgba(16,60,48,.04), 0 12rpx 40rpx rgba(16,60,48,.06); }
.ai { width: 44rpx; height: 44rpx; border-radius: 22rpx; background: linear-gradient(120deg, #34C9A3, #62E17D); display: flex; align-items: center; justify-content: center; } .ai image { width: 28rpx; height: 28rpx; }
.mc { padding-top: 12rpx; padding-bottom: 12rpx; }
.mn { margin-left: 24rpx; }
.ck { height: 60rpx; padding: 0 28rpx; border-radius: 30rpx; font-size: 24rpx; font-weight: 600; display: flex; align-items: center; gap: 6rpx; }
.ck image { width: 24rpx; height: 24rpx; }
.ck.done { color: #31CD4C; background: #EAF9EC; }
.ck.todo { color: #fff; background: linear-gradient(120deg, #34C9A3, #62E17D); box-shadow: 0 8rpx 20rpx -4rpx rgba(16,196,149,.5); }
.mk { margin-left: 8rpx; background: #F2F4F3; color: #666; padding: 2rpx 12rpx; border-radius: 12rpx; font-size: 20rpx; font-weight: 400; }
.ck.mkb { color: #10C495; background: #E3F8F1; }
.ck.sk { color: #888; background: #EEF0EF; font-weight: 400; }
.acts { display: flex; align-items: center; gap: 4rpx; }
.skb { font-size: 24rpx; color: #888; padding: 12rpx 16rpx; }
.miss { display: flex; align-items: center; gap: 8rpx; margin-top: 4rpx; padding: 24rpx 0 12rpx; border-top: 1rpx solid #F0F2F1; }
.vt { position: relative; margin: 0 32rpx 24rpx; padding: 32rpx; border-radius: 32rpx; background: #fff; box-shadow: 0 2rpx 4rpx rgba(16,60,48,.04), 0 12rpx 40rpx rgba(16,60,48,.06); }
.vh { display: flex; align-items: center; } .vtx { margin-left: 24rpx; }
.vsub { margin: 12rpx 0 0 104rpx; }
.vbtn { margin-top: 28rpx; }
.vc { display: flex; align-items: center; }
.mini { display: flex; gap: 24rpx; margin: 0 32rpx 24rpx; }
.mini .card { margin: 0; flex: 1; display: flex; gap: 20rpx; align-items: center; padding: 28rpx; }
</style>
