<template>
  <view class="pg">
    <view class="safe-top"></view>
    <view class="deco"></view>
    <image class="logo" src="/static/brand/logo.png" />
    <view class="h1">小压管家</view>
    <view class="sub f14 c6">记录血压 · 提醒用药 · 家人共享</view>
    <view class="card">
      <view class="pt" v-for="p in points" :key="p.t">
        <view class="ic" :class="p.bg"><image :src="'/static/icons/' + p.i + '.svg'" /></view>
        <view class="f14 c3">{{ p.t }}</view>
      </view>
    </view>
    <view class="agree" @tap="agree = !agree">
      <view class="box" :class="{ on: agree }"><image v-if="agree" src="/static/icons/check-w.svg" /></view>
      <view class="f12 c6 flex1">我同意收集和保存我的血压、心率、身体感受和用药信息，仅供本人及我授权的家人查看。<text class="lk" @tap.stop="show = true">《健康数据隐私说明》</text></view>
    </view>
    <view class="pad"><view class="btn" :class="{ dis: !agree }" @tap="go">微信一键登录</view></view>
    <view class="tip f12 c9">{{ agree ? '登录后会为你准备一些示例数据，可在「我的」中清除' : '请先勾选上方同意，才能开始使用' }}</view>

    <view class="mask" v-if="show" @tap="show = false">
      <view class="modal" @tap.stop>
        <view class="f16 c0 b mt">健康数据隐私说明</view>
        <scroll-view scroll-y class="mbody f12 c6">
          <view v-for="(t, i) in privacy" :key="i" class="pp">{{ t }}</view>
        </scroll-view>
        <view class="mbtns">
          <view class="f16 c6 mb" @tap="show = false">关闭</view>
          <view class="f16 cg b mb" @tap="agree = true; show = false">同意并勾选</view>
        </view>
      </view>
    </view>
  </view>
</template>
<script>
import { hasConsent, isLoggedIn, login, getRole, homeOf, syncProfile } from '../../api.js'
import { PRIVACY } from '../../common/privacy.js'
export default {
  data: () => ({ agree: false, show: false, privacy: PRIVACY, points: [
    { i: 'edit-g', bg: 'bg-g', t: '几秒记下血压和心率' }, { i: 'bell-b', bg: 'bg-b', t: '按时提醒测量和用药' }, { i: 'family-o', bg: 'bg-o', t: '绑定家人，一起查看记录' }] }),
  onLoad() { if (isLoggedIn() && hasConsent()) this.next() },
  methods: {
    async next(synced) { if (!synced) await syncProfile(); const r = getRole(); uni.reLaunch({ url: r ? homeOf(r) : '/pages/role/role' }) },
    async go() {
      if (!this.agree) return uni.showToast({ title: '请先勾选同意', icon: 'none' })
      uni.showLoading({ title: '登录中' })
      try { await login(); uni.hideLoading(); this.next(true) } catch (e) { uni.hideLoading(); uni.showToast({ title: '登录失败，请重试', icon: 'none' }) }
    }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: linear-gradient(180deg, #D7F7EA 0%, #EEF9F4 40%, #F5F7F6 70%); position: relative; overflow: hidden; padding-bottom: 60rpx; }
.deco { position: absolute; top: -120rpx; right: -160rpx; width: 520rpx; height: 520rpx; border-radius: 50%; background: radial-gradient(circle, rgba(167,255,185,.55), rgba(112,232,201,0) 70%); }
.logo { position: relative; display: block; width: 176rpx; height: 176rpx; border-radius: 40rpx; margin: 100rpx auto 36rpx; box-shadow: 0 32rpx 68rpx -12rpx rgba(16,196,149,.45); }
.h1 { text-align: center; font-size: 40rpx; font-weight: 700; color: #000; }
.sub { text-align: center; margin: 8rpx 0 64rpx; }
.pt { display: flex; align-items: center; gap: 24rpx; } .pt + .pt { margin-top: 24rpx; }
.agree { display: flex; gap: 20rpx; margin: 40rpx 48rpx 40rpx; align-items: flex-start; line-height: 1.6; }
.box { flex: none; width: 36rpx; height: 36rpx; border-radius: 10rpx; border: 3rpx solid #C4CBC8; background: #fff; margin-top: 2rpx; display: flex; align-items: center; justify-content: center; }
.box.on { background: #10C495; border-color: #10C495; } .box image { width: 28rpx; height: 28rpx; }
.lk { color: #10C495; font-weight: 600; }
.pad { padding: 0 32rpx; }
.tip { text-align: center; margin-top: 24rpx; }
.mask { position: fixed; inset: 0; background: rgba(0,0,0,.45); z-index: 99; display: flex; align-items: center; justify-content: center; }
.modal { width: 620rpx; background: #fff; border-radius: 32rpx; overflow: hidden; }
.mt { text-align: center; padding: 40rpx 0 16rpx; }
.mbody { max-height: 640rpx; padding: 0 40rpx; box-sizing: border-box; line-height: 1.7; }
.pp { margin-bottom: 16rpx; }
.mbtns { display: flex; border-top: 1rpx solid #EEF1F0; margin-top: 16rpx; }
.mb { flex: 1; text-align: center; height: 100rpx; line-height: 100rpx; } .mb + .mb { border-left: 1rpx solid #EEF1F0; }
</style>
