<template>
  <view class="pg">
    <view class="card"><view class="f16 c0 b h">健康数据隐私说明</view><view class="f14 c3 p" v-for="(t, i) in P" :key="i">{{ t }}</view></view>
    <view class="card" v-if="c"><view class="f12 c6">你于 {{ ft(c.agreedAt) }} 同意了 {{ c.version }} 版说明</view></view>
  </view>
</template>
<script>
import { PRIVACY } from '../../common/privacy.js'
import { getConsent, fmtTime } from '../../api.js'
export default { data: () => ({ P: PRIVACY, c: getConsent() }), methods: { ft: t => new Date(t).getFullYear() + '年' + fmtTime(t) } }
</script>
<style scoped>.pg { min-height: 100vh; background: #F5F7F6; padding-top: 24rpx; } .h { margin-bottom: 20rpx; } .p { line-height: 1.8; margin-bottom: 16rpx; }</style>
