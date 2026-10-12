<template>
  <view class="pg">
    <view class="safe-top"></view>
    <view class="deco"></view>
    <view class="h1">你想怎么使用？</view>
    <view class="sub f14 c6">之后可以在「我的」里随时切换，两种身份可以同时使用</view>
    <view class="opt" :class="{ on: role === 'patient' }" @tap="pick('patient')">
      <view class="ic bg-g big"><image src="/static/icons/heart-g.svg" /></view>
      <view class="flex1 tx"><view class="f20 c0 b">我是患者</view><view class="f14 c6">记录我自己的血压和用药</view></view>
      <image class="arrow" src="/static/icons/right-9.svg" />
    </view>
    <view class="opt" :class="{ on: role === 'family' }" @tap="pick('family')">
      <view class="ic bg-o big"><image src="/static/icons/family-o.svg" /></view>
      <view class="flex1 tx"><view class="f20 c0 b">我是家人</view><view class="f14 c6">关注爸妈等家人的记录，只能查看</view></view>
      <image class="arrow" src="/static/icons/right-9.svg" />
    </view>
  </view>
</template>
<script>
import { guard, getRole, setRole } from '../../api.js'
export default {
  data: () => ({ role: '' }),
  onLoad() { if (guard()) this.role = getRole() },
  methods: {
    pick(r) { setRole(r); this.role = r; r === 'family' ? uni.reLaunch({ url: '/pages/family/home' }) : uni.switchTab({ url: '/pages/index/index' }) }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: linear-gradient(180deg, #D7F7EA 0%, #EEF9F4 40%, #F5F7F6 70%); position: relative; overflow: hidden; padding: 0 32rpx 60rpx; }
.deco { position: absolute; top: -120rpx; right: -160rpx; width: 520rpx; height: 520rpx; border-radius: 50%; background: radial-gradient(circle, rgba(167,255,185,.55), rgba(112,232,201,0) 70%); }
.h1 { position: relative; font-size: 40rpx; font-weight: 700; color: #000; margin: 120rpx 8rpx 12rpx; }
.sub { position: relative; margin: 0 8rpx 56rpx; }
.opt { position: relative; display: flex; align-items: center; background: #fff; border-radius: 32rpx; padding: 40rpx 32rpx; margin-bottom: 28rpx; box-shadow: 0 2rpx 4rpx rgba(16,60,48,.04), 0 12rpx 40rpx rgba(16,60,48,.06); }
.opt.on { box-shadow: inset 0 0 0 3rpx #10C495, 0 12rpx 40rpx rgba(16,60,48,.06); }
.big { width: 112rpx; height: 112rpx; border-radius: 32rpx; } .big image { width: 60rpx; height: 60rpx; }
.tx { margin-left: 28rpx; }
</style>
