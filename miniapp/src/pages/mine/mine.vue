<template>
  <view class="pg">
    <view class="deco"></view>
    <view class="safe-top"></view>
    <view class="navh f20 c0 b">我的</view>
    <view class="hero">
      <view class="ring"></view>
      <view class="top">
        <view class="av"><image src="/static/icons/user-w.svg" /></view>
        <view class="flex1"><view class="f20 cw b">{{ name }}</view><view class="f12 cw o8">坚持记录第 {{ days }} 天</view></view>
        <view class="badge f10">隐私协议 v{{ ver }}</view>
      </view>
      <view class="stats">
        <view class="st"><view class="num f20 cw">{{ total }}</view><view class="f10 cw o8">测量次数</view></view>
        <view class="st"><view class="num f20 cw">{{ days }}</view><view class="f10 cw o8">记录天数</view></view>
        <view class="st"><view class="num f20 cw">{{ meds }}</view><view class="f10 cw o8">在用药品</view></view>
      </view>
    </view>
    <view class="sh f16 c0 b">常用功能</view>
    <view class="tiles">
      <view class="tile" v-for="e in g1" :key="e.t" @tap="go(e.u)">
        <view class="ti" :class="e.bg"><image :src="'/static/icons/' + e.i + '.svg'" /></view>
        <view class="f14 c0 b">{{ e.t }}</view><view class="f10 c9">{{ e.s }}</view>
      </view>
    </view>
    <view class="sh f16 c0 b">身份</view>
    <view class="card grp">
      <view class="cell" @tap="toFamily"><view class="ic sm bg-o"><image src="/static/icons/eye-o.svg" /></view><view class="flex1 lt"><view class="f14 c3">切换为「我是家人」</view><view class="f10 c9">{{ pats ? '正在关注 ' + pats + ' 位家人' : '输入邀请码，只读查看家人的记录' }}</view></view><image class="arrow" src="/static/icons/right-9.svg" /></view>
    </view>
    <view class="sh f16 c0 b">数据与隐私</view>
    <view class="card grp">
      <view class="cell" @tap="go('/pages/report/report')"><view class="ic sm bg-g"><image src="/static/icons/chart-g.svg" /></view><view class="flex1 lt"><view class="f14 c3">就诊报告卡</view><view class="f10 c9">生成一张图片，看病时给医生看</view></view><image class="arrow" src="/static/icons/right-9.svg" /></view>
      <view class="cell" @tap="go('/pages/privacy/privacy')"><view class="ic sm bg-g"><image src="/static/icons/shield-g.svg" /></view><view class="flex1 lt"><view class="f14 c3">隐私说明</view><view class="f10 c9">数据怎么存、谁能看</view></view><image class="arrow" src="/static/icons/right-9.svg" /></view>
      <view class="cell" @tap="exp"><view class="ic sm bg-g"><image src="/static/icons/download-g.svg" /></view><view class="flex1 lt"><view class="f14 c3">导出数据</view><view class="f10 c9">下载全部记录（JSON 文件）</view></view><image class="arrow" src="/static/icons/right-9.svg" /></view>
    </view>
    <view class="danger f14" @tap="logout"><image src="/static/icons/trash-6.svg" />删除全部数据并注销</view>
    <view class="f10 c9 ft">本工具仅用于记录，不提供诊断建议</view>

    <view class="mask" v-if="json" @tap="json = ''"><view class="modal" @tap.stop><view class="f16 c0 b mt">导出数据</view><scroll-view scroll-y class="jb"><text class="f10 c6" user-select>{{ json }}</text></scroll-view><view class="mb f16 cg b" @tap="json = ''">完成</view></view></view>
  </view>
</template>
<script>
import { getVisit, listPatients, setRole, guard, getProfile, listRecords, listMedDefs, listFamily, getRemind, exportData, deleteAccount, getConsent, dayKey } from '../../api.js'
export default {
  data: () => ({ visit: '', pats: 0, name: '', total: 0, days: 0, meds: 0, fam: 0, rem: '', json: '', ver: '' }),
  computed: { g1() { return [
    { t: '用药管理', i: 'pill-g', bg: 'bg-g', u: '/pages/meds/meds', s: this.meds ? this.meds + ' 种药' : '未添加' },
    { t: '测量提醒', i: 'bell-b', bg: 'bg-b', u: '/pages/remind/remind', s: this.rem },
    { t: '家人共享', i: 'family-o', bg: 'bg-o', u: '/pages/family/family', s: this.fam ? this.fam + ' 位家人' : '未绑定' }] } },
  async onShow() {
    if (!guard()) return
    this.name = getProfile().nickname; this.ver = (getConsent() || {}).version
    const r = await listRecords(); this.total = r.length; this.days = new Set(r.map(x => dayKey(x.time))).size
    this.meds = (await listMedDefs()).length; this.fam = (await listFamily()).length
    const v = await getVisit(); this.visit = v ? v.date.slice(5).replace('-', '/') + ' 复诊' : '未设置'
    this.pats = (await listPatients().catch(() => [])).length
    const m = await getRemind(); const n = [m.morning.on, m.evening.on].filter(Boolean).length; this.rem = n ? '已开启 ' + n + ' 个' : '未开启'
  },
  methods: {
    go(url) { uni.navigateTo({ url }) },
    toFamily() { setRole('family'); uni.reLaunch({ url: '/pages/family/home' }) },
    async exp() {
      const s = JSON.stringify(await exportData(), null, 2)
      // #ifdef H5
      const a = document.createElement('a'); a.href = URL.createObjectURL(new Blob([s], { type: 'application/json' }))
      a.download = 'bp-data-' + new Date().toISOString().slice(0, 10) + '.json'; document.body.appendChild(a); a.click(); a.remove()
      uni.showToast({ title: '已下载 JSON 文件', icon: 'none' })
      // #endif
      // #ifndef H5
      this.json = s
      // #endif
    },
    logout() { uni.showModal({ title: '删除全部数据并注销', content: '你的所有记录、用药、提醒和家人绑定都会被删除，且无法恢复。确定继续吗？', confirmText: '删除', confirmColor: '#E5484D', success: async r => { if (r.confirm) { await deleteAccount(); uni.reLaunch({ url: '/pages/login/login' }) } } }) }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: linear-gradient(180deg, #DDF8EE 0, #EEF9F4 300rpx, #F5F7F6 560rpx); position: relative; overflow: hidden; padding-bottom: 24rpx; }
.deco { position: absolute; top: -120rpx; right: -160rpx; width: 520rpx; height: 520rpx; border-radius: 50%; background: radial-gradient(circle, rgba(167,255,185,.55), rgba(112,232,201,0) 70%); }
.cw { color: #fff; } .o8 { opacity: .85; }
.hero { position: relative; overflow: hidden; margin: 8rpx 32rpx 40rpx; padding: 36rpx 36rpx 28rpx; border-radius: 40rpx; background: linear-gradient(120deg, #10C495 0%, #34C9A3 45%, #62E17D 100%); box-shadow: 0 24rpx 56rpx -16rpx rgba(16,196,149,.55); }
.ring { position: absolute; right: -60rpx; top: -100rpx; width: 340rpx; height: 340rpx; border-radius: 50%; border: 52rpx solid rgba(255,255,255,.09); }
.top { position: relative; display: flex; align-items: center; gap: 24rpx; }
.av { width: 112rpx; height: 112rpx; border-radius: 56rpx; background: rgba(255,255,255,.22); border: 4rpx solid rgba(255,255,255,.55); display: flex; align-items: center; justify-content: center; }
.av image { width: 60rpx; height: 60rpx; }
.badge { color: #fff; background: rgba(255,255,255,.2); padding: 6rpx 16rpx; border-radius: 20rpx; }
.stats { position: relative; display: flex; margin-top: 36rpx; padding-top: 24rpx; border-top: 1rpx solid rgba(255,255,255,.3); }
.st { flex: 1; text-align: center; } .st + .st { border-left: 1rpx solid rgba(255,255,255,.25); }
.num { font-family: "DIN Alternate", "D-DIN-EXP", "Helvetica Neue", Arial, sans-serif; font-weight: 700; line-height: 1.3; }
.sh { position: relative; margin: 0 40rpx 20rpx; }
.tiles { display: flex; gap: 20rpx; margin: 0 32rpx 40rpx; }
.tile { flex: 1; background: #fff; border-radius: 32rpx; padding: 28rpx 20rpx 24rpx; display: flex; flex-direction: column; align-items: center; gap: 4rpx; box-shadow: 0 2rpx 4rpx rgba(16,60,48,.04), 0 12rpx 40rpx rgba(16,60,48,.05); }
.ti { width: 88rpx; height: 88rpx; border-radius: 28rpx; display: flex; align-items: center; justify-content: center; margin-bottom: 12rpx; } .ti image { width: 48rpx; height: 48rpx; }
.grp { padding-top: 4rpx; padding-bottom: 4rpx; }
.ic.sm { width: 64rpx; height: 64rpx; border-radius: 20rpx; } .ic.sm image { width: 36rpx; height: 36rpx; }
.lt { margin-left: 24rpx; }
.arrow { margin-left: 8rpx; }
.danger { margin: 16rpx 32rpx 0; height: 96rpx; border-radius: 48rpx; background: #fff; color: #666; display: flex; align-items: center; justify-content: center; gap: 12rpx; box-shadow: 0 12rpx 40rpx rgba(16,60,48,.05); }
.danger image { width: 32rpx; height: 32rpx; }
.ft { text-align: center; margin: 28rpx 0 40rpx; }
.mask { position: fixed; inset: 0; background: rgba(0,0,0,.45); z-index: 99; display: flex; align-items: center; justify-content: center; }
.modal { width: 640rpx; background: #fff; border-radius: 32rpx; overflow: hidden; }
.mt { text-align: center; padding: 36rpx 0 16rpx; } .jb { height: 700rpx; padding: 0 32rpx; box-sizing: border-box; white-space: pre-wrap; word-break: break-all; }
.mb { text-align: center; height: 100rpx; line-height: 100rpx; border-top: 1rpx solid #EEF1F0; }
</style>
