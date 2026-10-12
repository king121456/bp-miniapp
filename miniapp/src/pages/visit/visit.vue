<template>
  <view class="pg">
    <!-- 即将到来的复诊 -->
    <view class="hero" v-if="up && !editing">
      <view class="ring"></view>
      <view class="f12 o9">下次复诊</view>
      <view class="dt"><text class="num big">{{ md(up.date) }}</text><text class="f16 o9 lft">{{ left(up.date) }}</text></view>
      <view class="f14 o9" v-if="up.hospital || up.department">{{ [up.hospital, up.department].filter(Boolean).join(' · ') }}</view>
      <view class="f12 o9 nt" v-if="up.note">备注：{{ up.note }}</view>
      <view class="f12 o9 rm"><image src="/static/icons/bell-w.svg" />{{ remindStr }}</view>
      <view class="hb" @tap="toCard"><image src="/static/icons/card-g.svg" />生成就诊卡片</view>
    </view>
    <view class="card emp" v-if="!up && !editing">
      <view class="ic bg-b ei"><image src="/static/icons/calendar-b.svg" /></view>
      <view class="f16 c0 b">还没有记下复诊日期</view>
      <view class="f12 c9">记下医生约的下次复诊日期，前一天会提醒你</view>
    </view>
    <view class="acts" v-if="!editing">
      <block v-if="up"><view class="btn ghost flex1" @tap="cancel">取消这次复诊</view><view class="btn flex1" @tap="edit(true)">修改</view></block>
      <view v-else class="btn flex1" @tap="edit(false)">记下复诊日期</view>
    </view>
    <view class="lk f14 cg" v-if="up && !editing" @tap="edit(false)">记下新的复诊日期</view>

    <!-- 表单 -->
    <view class="card" v-if="editing">
      <view class="f16 c0 b fh">{{ form.id ? '修改复诊' : '记下复诊日期' }}</view>
      <picker mode="date" :value="form.date" :start="today" :end="maxDate" @change="e => form.date = e.detail.value">
        <view class="cell"><text class="f14 c3 flex1">复诊日期</text><text class="f16 num" :class="form.date ? 'c0' : 'c9'">{{ form.date ? md(form.date) + ' 星期' + wk(form.date) : '请选择' }}</text><image class="arrow" src="/static/icons/right-9.svg" /></view>
      </picker>
      <view class="cell"><text class="f14 c3">医院</text><input class="ip f14 c0" v-model="form.hospital" maxlength="30" placeholder="选填" placeholder-class="phc" /></view>
      <view class="cell"><text class="f14 c3">科室</text><input class="ip f14 c0" v-model="form.department" maxlength="20" placeholder="选填" placeholder-class="phc" /></view>
      <view class="cell"><text class="f14 c3">备注</text><input class="ip f14 c0" v-model="form.note" maxlength="100" placeholder="选填，仅自己可见" placeholder-class="phc" /></view>
      <picker mode="time" :value="form.remindTime" @change="e => form.remindTime = e.detail.value">
        <view class="cell"><text class="f14 c3 flex1">提醒时间</text><text class="f14 c0">复诊前一天 <text class="num">{{ form.remindTime }}</text></text><image class="arrow" src="/static/icons/right-9.svg" /></view>
      </picker>
      <view class="f10 c9 tip">{{ local ? '演示版：到提醒时间后在首页显示复诊待办。' : '保存时会请求微信提醒授权；不能推送时，首页会显示复诊待办。' }}家人只能看到日期、医院和科室，看不到备注。</view>
      <view class="acts in"><view class="btn ghost flex1" @tap="editing = false">取消</view><view class="btn flex1" :class="{ dis: !form.date }" @tap="save">保存复诊日期</view></view>
    </view>

    <!-- 历史 -->
    <view class="sh"><text class="t">复诊历史</text><text class="f12 c9">{{ hist.length }} 次</text></view>
    <view class="card lst">
      <view v-if="!hist.length" class="empty">暂无复诊历史</view>
      <view class="cell" v-for="v in hist" :key="v.id">
        <view class="flex1"><view class="f14 c0 b">{{ v.date.replace(/-/g, '.') }}<text class="hs">{{ v.status === 'past' ? '已过' : '已取消' }}</text></view>
          <view class="f12 c9">{{ [v.hospital, v.department].filter(Boolean).join(' · ') || '未填医院' }}{{ v.note ? ' · ' + v.note : '' }}</view></view>
        <view class="ub f12 c6" @tap="del(v)">删除</view>
      </view>
    </view>
    <view class="f10 c9 ft">复诊日期由你自己填写，本工具不提供就医建议</view>
  </view>
</template>
<script>
import { guard, USE_LOCAL, getVisit, listVisitHistory, saveVisit, cancelVisit, deleteVisit, daysUntil, visitRemindAt, VISIT_REMIND_DEFAULT, dismissAskNextVisit, fmtTime, WEEK } from '../../api.js'
const p2 = n => String(n).padStart(2, '0')
const ymd = d => d.getFullYear() + '-' + p2(d.getMonth() + 1) + '-' + p2(d.getDate())
const pd = s => { const [y, m, d] = s.split('-').map(Number); return new Date(y, m - 1, d) }
export default {
  data() { const t = new Date(); return { up: null, hist: [], editing: false, form: {}, local: USE_LOCAL, today: ymd(t), maxDate: ymd(new Date(t.getFullYear() + 1, t.getMonth(), t.getDate())) } },
  computed: {
    remindStr() { const v = this.up; if (!v) return ''; const at = visitRemindAt(v); return at > Date.now() ? '将于 ' + fmtTime(at) + ' 提醒' : '已在首页显示复诊待办' }
  },
  onLoad(o) { if (o && o.new) this.$nextTick(() => this.edit(false)) },
  onShow() { if (guard()) { dismissAskNextVisit(); this.load() } },
  methods: {
    async load() { [this.up, this.hist] = await Promise.all([getVisit(), listVisitHistory()]) },
    md(s) { const d = pd(s); return `${d.getMonth() + 1}月${d.getDate()}日` }, wk(s) { return WEEK[pd(s).getDay()] },
    left(s) { const n = daysUntil(s); return n === 0 ? '今天' : n === 1 ? '明天' : '还有 ' + n + ' 天' },
    edit(modify) { const v = modify && this.up; this.form = v ? { ...v } : { date: '', hospital: (this.up || this.hist[0] || {}).hospital || '', department: (this.up || this.hist[0] || {}).department || '', note: '', remindTime: VISIT_REMIND_DEFAULT }; this.editing = true },
    async save() {
      const f = this.form
      if (!f.date) return uni.showToast({ title: '请选择复诊日期', icon: 'none' })
      if (f.date < this.today || f.date > this.maxDate) return uni.showToast({ title: '请选择今天到一年内的日期', icon: 'none' })
      const go = async replace => {
        try { await saveVisit(f, replace); this.editing = false; uni.showToast({ title: '已保存', icon: 'none' }); this.load() }
        catch (e) {
          if (e && e.conflict) return uni.showModal({ title: '更换复诊日期', content: `已有 ${this.md(e.conflict.date)} 的复诊，要改成新的日期吗？`, confirmText: '改成新的', confirmColor: '#10C495', success: r => r.confirm && go(true) })
          uni.showToast({ title: (e && (e.message || e.errMsg)) || '保存失败', icon: 'none' })
        }
      }
      go(false)
    },
    cancel() { uni.showModal({ title: '取消复诊', content: `取消 ${this.md(this.up.date)} 的复诊后不再提醒，记录会留在复诊历史中。`, confirmText: '确定取消', cancelText: '不取消', confirmColor: '#10C495', success: async r => { if (r.confirm) { await cancelVisit(this.up.id); this.load() } } }) },
    del(v) { uni.showModal({ title: '删除记录', content: '从复诊历史中删除这条记录？', confirmColor: '#10C495', success: async r => { if (r.confirm) { await deleteVisit(v.id); this.load() } } }) },
    toCard() { uni.navigateTo({ url: '/pages/report/report?from=visit' }) }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: linear-gradient(180deg, #DDF8EE 0, #F5F7F6 400rpx); padding: 24rpx 0 48rpx; }
.o9 { opacity: .9; }
.hero { position: relative; overflow: hidden; margin: 0 32rpx 24rpx; padding: 36rpx; border-radius: 40rpx; color: #fff; background: linear-gradient(120deg, #10C495 0%, #34C9A3 45%, #62E17D 100%); box-shadow: 0 24rpx 56rpx -16rpx rgba(16,196,149,.55); }
.ring { position: absolute; right: -60rpx; top: -100rpx; width: 340rpx; height: 340rpx; border-radius: 50%; border: 52rpx solid rgba(255,255,255,.09); }
.dt { position: relative; display: flex; align-items: baseline; margin: 8rpx 0 8rpx; } .big { font-size: 80rpx; line-height: 1.1; } .lft { margin-left: 20rpx; }
.nt { margin-top: 8rpx; }
.rm { display: flex; align-items: center; gap: 8rpx; margin-top: 16rpx; } .rm image { width: 28rpx; height: 28rpx; }
.hb { position: relative; margin-top: 28rpx; height: 88rpx; border-radius: 44rpx; background: #fff; color: #10C495; font-size: 32rpx; font-weight: 600; display: flex; align-items: center; justify-content: center; gap: 12rpx; } .hb image { width: 40rpx; height: 40rpx; }
.emp { text-align: center; padding: 48rpx 32rpx; } .ei { margin: 0 auto 20rpx; }
.acts { display: flex; gap: 24rpx; margin: 0 32rpx 24rpx; } .acts.in { margin: 32rpx 0 0; }
.lk { text-align: center; margin: 0 0 32rpx; }
.fh { margin-bottom: 8rpx; }
.ip { flex: 1; text-align: right; margin-left: 24rpx; }
.tip { margin-top: 12rpx; line-height: 1.6; }
.lst { padding-top: 8rpx; padding-bottom: 8rpx; }
.hs { margin-left: 12rpx; background: #F2F4F3; color: #666; padding: 2rpx 12rpx; border-radius: 12rpx; font-size: 20rpx; font-weight: 400; }
.ub { padding: 10rpx 20rpx; border-radius: 28rpx; background: #F2F4F3; }
.ft { text-align: center; margin-top: 24rpx; }
</style>
