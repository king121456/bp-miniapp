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
    <view class="sh"><text class="t">打卡记录</text><text class="f12 c9">近 30 天</text></view>
    <view class="card lst">
      <view v-if="!cks.length" class="empty">暂无打卡</view>
      <view class="cell" v-for="(c, i) in cks.slice(0, limit)" :key="i">
        <view class="dot"><image src="/static/icons/check-l.svg" /></view>
        <view class="flex1 mn"><view class="f14 c0">{{ c.name }}</view><view class="f10 c9">计划 {{ c.time }}</view></view>
        <text class="f12 c6">{{ dl(c.at) }} <text class="num">{{ hm(c.at) }}</text></text>
      </view>
      <view v-if="cks.length > limit" class="more f12 cg" @tap="limit += 20">查看更多</view>
    </view>
  </view>
</template>
<script>
import { guard, listMedDefs, listCheckins, dayLabel, hhmm } from '../../api.js'
export default {
  data: () => ({ meds: [], cks: [], limit: 10 }),
  async onShow() { if (!guard()) return; this.meds = await listMedDefs(); this.cks = await listCheckins() },
  methods: { edit(id) { uni.navigateTo({ url: '/pages/meds/edit' + (id ? '?id=' + id : '') }) }, dl: dayLabel, hm: hhmm }
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
.more { text-align: center; padding: 20rpx 0 12rpx; border-top: 1rpx solid #F0F2F1; }
</style>
