<template>
  <view class="page">
    <view class="logo"><image class="logo-i" src="/static/icons/heart.svg" /></view>
    <view class="title">血压记录</view>
    <view class="sub muted">记录血压 · 提醒用药 · 家人共享</view>
    <view class="card">
      <view class="pt" v-for="p in points" :key="p.t"><view class="ic"><image :src="p.i" class="ii" /></view><text>{{ p.t }}</text></view>
    </view>
    <view class="agree" @tap="ok = !ok">
      <view :class="['box', ok && 'on']"><text v-if="ok">✓</text></view>
      <view>我同意收集和保存我的血压、心率、症状和用药信息，仅供本人及授权家人查看。<text class="link" @tap.stop="privacy">隐私说明</text></view>
    </view>
    <view class="wrap"><view :class="['btn-primary', !ok && 'disabled']" @tap="start">微信一键登录</view></view>
    <view class="tip muted">{{ ok ? '登录后即可开始记录' : '请先勾选上方同意，才能开始使用' }}</view>
  </view>
</template>
<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { hasConsent, login } from '../../api.js'
const ok = ref(false)
const points = [{ i: '/static/icons/edit.svg', t: '三步记下血压和心率' }, { i: '/static/icons/alarm.svg', t: '按时提醒测量和用药' }, { i: '/static/icons/family.svg', t: '绑定家人，一起查看记录' }]
onLoad(() => { if (hasConsent()) uni.switchTab({ url: '/pages/index/index' }) })
const privacy = () => uni.showModal({ title: '隐私说明', content: '你的健康数据只用于记录和展示，不做诊断，不提供给第三方。你可以随时在"我的"里导出或删除全部数据。', showCancel: false, confirmText: '知道了' })
const start = async () => {
  if (!ok.value) return uni.showToast({ title: '请先勾选同意', icon: 'none' })
  try { await login(); uni.switchTab({ url: '/pages/index/index' }) } catch (e) { uni.showToast({ title: '登录没成功，请再试一次', icon: 'none' }) }
}
</script>
<style lang="scss">
.page{min-height:100vh;background:linear-gradient(180deg,#D2F5E8 0%,$c-bg 55%);padding-top:100rpx}
.logo{width:192rpx;height:192rpx;border-radius:56rpx;margin:0 auto 40rpx;background:linear-gradient(140deg,#34C9A3,#0EA884);display:flex;align-items:center;justify-content:center;box-shadow:0 32rpx 68rpx rgba(16,196,149,.35)}
.logo-i{width:100rpx;height:100rpx}
.title{text-align:center;font-size:64rpx;font-weight:700}.sub{text-align:center;margin:16rpx 0 80rpx}
.pt{display:flex;align-items:center;gap:28rpx;margin-bottom:28rpx}.pt:last-child{margin:0}
.ic{width:88rpx;height:88rpx;border-radius:24rpx;background:$c-primary-bg;display:flex;align-items:center;justify-content:center}.ii{width:48rpx;height:48rpx}
.agree{display:flex;gap:24rpx;margin:40rpx 48rpx 44rpx;font-size:34rpx;line-height:1.6}
.box{flex:none;width:56rpx;height:56rpx;border-radius:16rpx;border:4rpx solid $c-icon;background:#fff;color:#fff;display:flex;align-items:center;justify-content:center;font-size:36rpx}
.box.on{background:$c-primary;border-color:$c-primary}.link{color:$c-primary;font-weight:600}
.wrap{padding:0 $gap}.tip{text-align:center;font-size:$fs-aux;margin-top:28rpx}
</style>
