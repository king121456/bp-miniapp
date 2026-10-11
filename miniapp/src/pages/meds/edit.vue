<template>
  <view class="pg">
    <view class="card">
      <view class="fl"><text class="f14 c0 b lb">药品名称</text><input class="ip f14" v-model="m.name" placeholder="如：氨氯地平片" placeholder-class="phc" maxlength="20" /></view>
      <view class="fl"><text class="f14 c0 b lb">剂量</text><input class="ip f14" v-model="m.dose" placeholder="如：5mg · 1片" placeholder-class="phc" maxlength="20" /></view>
    </view>
    <view class="sh"><text class="t">服药时间</text><text class="f12 c9">每天 {{ m.times.length }} 次</text></view>
    <view class="card">
      <view class="cell" v-for="(t, i) in m.times" :key="i">
        <view class="ic bg-g sm"><image src="/static/icons/clock-g.svg" /></view>
        <picker mode="time" :value="t" class="flex1" @change="e => m.times.splice(i, 1, e.detail.value)"><view class="f20 c0 num pk">{{ t }}</view></picker>
        <text class="f12 c9 rm" @tap="m.times.splice(i, 1)">移除</text>
      </view>
      <picker mode="time" value="12:00" @change="e => addTime(e.detail.value)"><view class="addt f14 cg"><image src="/static/icons/plus-g.svg" />添加时间</view></picker>
    </view>
    <view class="pad"><view class="btn" @tap="save">保存</view><view v-if="m.id" class="del f14 c6" @tap="del">删除这个药品</view></view>
  </view>
</template>
<script>
import { listMedDefs, saveMed, deleteMed } from '../../api.js'
export default {
  data: () => ({ m: { id: '', name: '', dose: '', times: ['08:00'] } }),
  async onLoad(q) {
    if (q && q.id) { const f = (await listMedDefs()).find(x => x.id === q.id); if (f) this.m = { ...f, times: [...f.times] } }
    else uni.setNavigationBarTitle({ title: '添加药品' })
  },
  methods: {
    addTime(v) { if (this.m.times.includes(v)) return uni.showToast({ title: '这个时间已添加', icon: 'none' }); this.m.times.push(v); this.m.times.sort() },
    async save() {
      if (!this.m.name.trim()) return uni.showToast({ title: '请填写药品名称', icon: 'none' })
      if (!this.m.times.length) return uni.showToast({ title: '请至少添加一个服药时间', icon: 'none' })
      await saveMed({ ...this.m, name: this.m.name.trim(), dose: this.m.dose.trim(), times: [...new Set(this.m.times)] })
      uni.showToast({ title: '已保存', icon: 'success' }); setTimeout(() => uni.navigateBack(), 600)
    },
    del() { uni.showModal({ title: '删除药品', content: '删除「' + this.m.name + '」后，将不再出现在今日用药中。', confirmColor: '#10C495', success: async r => { if (r.confirm) { await deleteMed(this.m.id); uni.navigateBack() } } }) }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: #F5F7F6; padding-top: 24rpx; }
.fl { display: flex; align-items: center; height: 96rpx; } .fl + .fl { border-top: 1rpx solid #F0F2F1; }
.lb { width: 160rpx; } .ip { flex: 1; height: 96rpx; color: #000; }
.ic.sm { width: 64rpx; height: 64rpx; border-radius: 20rpx; } .ic.sm image { width: 36rpx; height: 36rpx; }
.pk { margin-left: 24rpx; } .rm { padding: 12rpx; }
.addt { display: flex; align-items: center; justify-content: center; gap: 8rpx; height: 88rpx; border-top: 1rpx solid #F0F2F1; margin-top: 4rpx; font-weight: 600; } .addt image { width: 32rpx; height: 32rpx; }
.pad { padding: 16rpx 32rpx 48rpx; } .del { text-align: center; margin-top: 28rpx; }
</style>
<style>.phc { color: #999; }</style>
