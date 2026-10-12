<template>
  <view class="pg">
    <view class="bn f12"><image src="/static/icons/eye-o.svg" />{{ pid ? '只读查看「' + name + '」的记录，不能修改' : '这是家人看到的页面（只读，不显示备注）' }}</view>
    <view class="card vc" v-if="visit">
      <view class="ic bg-b"><image src="/static/icons/calendar-b.svg" /></view>
      <view class="flex1 mn"><view class="f16 c0 b">下次复诊：{{ md(visit.date) }}（{{ left }}）</view><view class="f12 c9" v-if="visit.hospital || visit.department">{{ [visit.hospital, visit.department].filter(Boolean).join(' · ') }}</view></view>
    </view>
    <view class="card hd" v-if="latest">
      <view class="f12 c6">最近一次 · {{ ft(latest.time) }}</view>
      <view class="row bp"><text class="num v">{{ latest.sys }}/{{ latest.dia }}</text><text class="f12 c9 u">mmHg</text><text class="f12 c6 hr">心率 {{ latest.hr || '--' }}</text></view>
    </view>
    <view class="tabs"><view v-for="(t, i) in ['血压', '服药']" :key="t" class="tb f14" :class="{ on: tab === i }" @tap="tab = i">{{ t }}</view></view>
    <bp-overview v-if="tab === 0" :records="records" readonly />
    <view v-else>
      <view v-if="msgs.length" class="sh"><text class="t">消息</text><text class="f12 c9">{{ msgs.length }} 条</text></view>
      <view v-if="msgs.length" class="card lst">
        <view class="cell" v-for="m in msgs" :key="m.id"><view class="ic sm bg-k"><image src="/static/icons/bell-6.svg" /></view>
          <view class="flex1 mn"><view class="f14 c3">{{ m.text }}</view><view class="f12 c9" v-for="n in m.makeupNotes" :key="n">{{ n }}</view></view></view>
      </view>
      <view class="sh"><text class="t">服药历史</text><text class="f12 c9">近 7 天</text></view>
      <med-history :slots="slots" readonly @pick="s => pick = s" />
    </view>
    <view class="unf f14 c6" v-if="pid" @tap="unfollow">取消关注</view>
    <slot-sheet :item="pick" readonly @close="pick = null" />
    <view style="height:40rpx"></view>
  </view>
</template>
<script>
import { guard, listRecords, getProfile, fmtTime, listPatientRecords, listPatientSlots, getPatientVisit, listSlots, getVisit, daysUntil, unfollowPatient, listFamilyMessages } from '../../api.js'
export default {
  data: () => ({ msgs: [], records: [], slots: [], visit: null, name: '', pid: '', bid: '', tab: 0, pick: null }),
  computed: { latest() { return this.records[0] }, left() { const n = daysUntil(this.visit.date); return n === 0 ? '今天' : n === 1 ? '明天' : '还有 ' + n + ' 天' } },
  async onLoad(o) {
    if (!guard()) return
    this.pid = o.pid || ''; this.bid = o.bid || ''
    if (this.pid) {
      this.name = decodeURIComponent(o.name || '家人'); uni.setNavigationBarTitle({ title: this.name })
      const [r, s, v] = await Promise.all([listPatientRecords(this.pid), listPatientSlots(this.pid).catch(() => []), getPatientVisit(this.pid).catch(() => null)])
      this.records = r; this.slots = s; this.visit = v
      this.msgs = await listFamilyMessages(this.pid).catch(() => [])
    } else { // 患者预览：按家人可见范围裁剪
      this.name = getProfile().nickname
      const [r, s, v] = await Promise.all([listRecords(), listSlots(), getVisit()])
      this.records = r.map(x => ({ ...x, note: '' })); this.slots = s.map(x => ({ ...x, canMakeup: false, canUndo: false })); this.visit = v && { date: v.date, hospital: v.hospital, department: v.department }
    }
  },
  methods: {
    ft: fmtTime, md(s) { const [, m, d] = s.split('-').map(Number); return `${m}月${d}日` },
    unfollow() { uni.showModal({ title: '取消关注', content: `取消后将不能再查看${this.name}的记录。`, confirmColor: '#10C495', success: async r => { if (r.confirm) { await unfollowPatient(this.bid || this.pid); uni.navigateBack() } } }) }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: #F5F7F6; padding-top: 16rpx; }
.bn { display: flex; align-items: center; gap: 12rpx; margin: 0 32rpx 24rpx; padding: 16rpx 24rpx; border-radius: 20rpx; background: #FFF5E0; color: #A06C00; } .bn image { width: 32rpx; height: 32rpx; flex: none; }
.vc { display: flex; align-items: center; } .mn { margin-left: 24rpx; }
.bp { align-items: baseline; margin-top: 8rpx; } .v { font-size: 80rpx; color: #000; line-height: 1.1; } .u { margin-left: 8rpx; } .hr { margin-left: auto; }
.tabs { display: flex; margin: 0 32rpx 24rpx; background: #fff; border-radius: 24rpx; padding: 6rpx; box-shadow: 0 12rpx 40rpx rgba(16,60,48,.05); }
.tb { flex: 1; text-align: center; height: 72rpx; line-height: 72rpx; border-radius: 20rpx; color: #666; } .tb.on { background: #E3F8F1; color: #10C495; font-weight: 600; }
.lst { padding-top: 8rpx; padding-bottom: 8rpx; } .ic.sm { width: 64rpx; height: 64rpx; border-radius: 20rpx; } .ic.sm image { width: 36rpx; height: 36rpx; }
.unf { text-align: center; margin: 24rpx 0 0; }
</style>
