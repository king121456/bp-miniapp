<template>
  <view class="pg">
    <view class="deco"></view>
    <view class="safe-top"></view>
    <view class="hd"><view class="f12 c6">家人视角 · 只读查看</view><view class="f20 c0 b">我关注的家人</view></view>
    <view v-if="!list.length" class="card empty">还没有关注家人，输入家人给你的邀请码即可查看</view>
    <view class="card pc" v-for="p in list" :key="p.id" @tap="open(p)">
      <view class="top"><view class="av">{{ p.name.slice(-1) }}</view>
        <view class="flex1 mn"><view class="f16 c0 b">{{ p.name }}</view><view class="f12 c9">{{ p.latest ? '最近一次 ' + ft(p.latest.time) : '还没有血压记录' }}</view></view>
        <image class="arrow" src="/static/icons/right-9.svg" /></view>
      <view class="kv">
        <view><view class="f10 c9">最近血压</view><view class="num f20 c0">{{ p.latest ? p.latest.sys + '/' + p.latest.dia : '--' }}</view></view>
        <view><view class="f10 c9">今日服药</view><view class="num f20 c0">{{ p.medTaken }}/{{ p.medTotal }}</view></view>
        <view><view class="f10 c9">下次复诊</view><view class="f14 c0 b vd">{{ p.visit ? md(p.visit.date) : '未设置' }}</view></view>
      </view>
    </view>
    <view class="sh"><text class="t">关注新的家人</text></view>
    <view class="card bd">
      <input class="code num" v-model="code" type="number" maxlength="6" placeholder="输入 6 位邀请码" placeholder-class="phc" />
      <view class="f10 c9 bt">绑定后你将能查看对方的血压与服药记录，仅可查看，不能修改。</view>
      <view class="btn" :class="{ dis: code.length !== 6 }" @tap="bind">确认关注</view>
    </view>
    <view class="sw f14 c6" @tap="switchRole"><image src="/static/icons/refresh-6.svg" />切换为「我是患者」</view>
  </view>
</template>
<script>
import { guard, listPatients, listPatientRecords, listPatientSlots, getPatientVisit, bindFamily, setRole, fmtTime } from '../../api.js'
export default {
  data: () => ({ list: [], code: '' }),
  onShow() { if (guard()) { setRole('family'); this.load() } },
  methods: {
    async load() {
      const ps = await listPatients(); const td = new Date(); const tk = td.getFullYear() + '-' + String(td.getMonth() + 1).padStart(2, '0') + '-' + String(td.getDate()).padStart(2, '0')
      this.list = await Promise.all(ps.map(async p => {
        const [recs, slots, visit] = await Promise.all([listPatientRecords(p.patientId).catch(() => []), listPatientSlots(p.patientId, 1).catch(() => []), getPatientVisit(p.patientId).catch(() => null)])
        const today = slots.filter(s => s.date === tk && s.status !== 'future')
        return { ...p, latest: recs[0], visit, medTotal: today.length, medTaken: today.filter(s => s.status === 'taken').length }
      }))
    },
    open(p) { uni.navigateTo({ url: '/pages/family/view?pid=' + p.patientId + '&name=' + encodeURIComponent(p.name) + '&bid=' + p.id }) },
    async bind() {
      if (this.code.length !== 6) return
      try { await bindFamily(this.code); this.code = ''; uni.showToast({ title: '已关注', icon: 'none' }); this.load() } catch (e) { uni.showToast({ title: (e && e.message) || '邀请码无效或已过期', icon: 'none' }) }
    },
    switchRole() { setRole('patient'); uni.reLaunch({ url: '/pages/index/index' }) },
    ft: fmtTime, md(s) { const [, m, d] = s.split('-').map(Number); return `${m}月${d}日` }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: linear-gradient(180deg, #DDF8EE 0, #EEF9F4 300rpx, #F5F7F6 560rpx); position: relative; overflow: hidden; padding-bottom: 48rpx; }
.deco { position: absolute; top: -120rpx; right: -160rpx; width: 520rpx; height: 520rpx; border-radius: 50%; background: radial-gradient(circle, rgba(167,255,185,.55), rgba(112,232,201,0) 70%); }
.hd { position: relative; padding: 16rpx 40rpx 32rpx; }
.pc { position: relative; }
.top { display: flex; align-items: center; }
.av { width: 88rpx; height: 88rpx; border-radius: 44rpx; background: #FFF5E0; color: #FBA900; font-size: 32rpx; font-weight: 600; display: flex; align-items: center; justify-content: center; flex: none; }
.mn { margin-left: 24rpx; }
.kv { display: flex; margin-top: 28rpx; padding-top: 24rpx; border-top: 1rpx solid #F0F2F1; } .kv > view { flex: 1; } .vd { line-height: 1.6; margin-top: 4rpx; }
.sh { position: relative; margin-top: 16rpx; }
.code { height: 96rpx; background: #F5F7F6; border-radius: 24rpx; text-align: center; font-size: 40rpx; letter-spacing: 8rpx; }
.bt { margin: 16rpx 0 24rpx; line-height: 1.6; }
.sw { display: flex; align-items: center; justify-content: center; gap: 12rpx; margin: 16rpx 32rpx 0; height: 96rpx; border-radius: 48rpx; background: #fff; box-shadow: 0 12rpx 40rpx rgba(16,60,48,.05); } .sw image { width: 32rpx; height: 32rpx; }
</style>
