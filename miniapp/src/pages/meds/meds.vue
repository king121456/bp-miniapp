<template>
  <view class="pg">
    <view class="sh top"><text class="t">我的药品</text><text class="f12 c9">{{ meds.length }} 种</text></view>
    <view class="card lst">
      <view v-if="!meds.length" class="empty">还没有添加药品</view>
      <view class="cell" v-for="m in meds" :key="m.id" @tap="edit(m.id)">
        <view class="ic bg-g"><image src="/static/icons/pill-g.svg" /></view>
        <view class="flex1 mn"><view class="f14 c0 b">{{ m.name }}</view><view class="f12 c9">{{ m.dose || '未填剂量' }} · 每天 {{ m.times.length }} 次</view>
          <view class="tms"><text class="tm num f12" v-for="t in m.times" :key="t">{{ t }}</text></view></view>
        <image class="arrow" src="/static/icons/right-9.svg" />
      </view>
    </view>
    <view class="pad"><view class="btn" @tap="edit('')">添加药品</view></view>
    <view class="sh"><text class="t">服药历史</text><text class="f12 c9">近 {{ days }} 天 · 7 天内未打卡可补记</text></view>
    <med-history :slots="slots" @pick="s => pick = s" />
    <view v-if="days < 30" class="more f12 cg" @tap="more">查看更早记录</view>
    <slot-sheet :item="pick" @close="pick = null" @done="reload" />
  </view>
</template>
<script>
import { guard, listMedDefs, listSlots } from '../../api.js'
export default {
  data: () => ({ meds: [], slots: [], days: 7, pick: null }),
  async onShow() { if (!guard()) return; this.meds = await listMedDefs(); this.reload() },
  methods: {
    edit(id) { uni.navigateTo({ url: '/pages/meds/edit' + (id ? '?id=' + id : '') }) },
    async reload() { this.pick = null; this.slots = await listSlots(this.days) },
    more() { this.days = 30; this.reload() }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: #F5F7F6; padding-bottom: 48rpx; }
.top { padding-top: 24rpx; }
.lst { padding-top: 8rpx; padding-bottom: 8rpx; }
.mn { margin-left: 24rpx; }
.tms { display: flex; flex-wrap: wrap; gap: 12rpx; margin-top: 8rpx; }
.tm { background: #E3F8F1; color: #10C495; padding: 2rpx 14rpx; border-radius: 16rpx; }
.pad { padding: 8rpx 32rpx 48rpx; }
.dot { width: 48rpx; height: 48rpx; border-radius: 24rpx; background: #EAF9EC; display: flex; align-items: center; justify-content: center; } .dot image { width: 28rpx; height: 28rpx; }
.more { text-align: center; padding: 8rpx 0 24rpx; }
</style>
