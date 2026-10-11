<template>
  <view class="pg">
    <view class="bn f12"><image src="/static/icons/eye-o.svg" />你正在以家人身份只读查看「{{ name }}」的记录</view>
    <view class="card hd" v-if="latest">
      <view class="f12 c6">最近一次 · {{ ft(latest.time) }}</view>
      <view class="row bp"><text class="num v">{{ latest.sys }}/{{ latest.dia }}</text><text class="f12 c9 u">mmHg</text><text class="f12 c6 hr">心率 {{ latest.hr || '--' }}</text></view>
    </view>
    <bp-overview :records="records" readonly />
    <view style="height:40rpx"></view>
  </view>
</template>
<script>
import { guard, listRecords, getProfile, fmtTime } from '../../api.js'
export default {
  data: () => ({ records: [], name: '' }),
  computed: { latest() { return this.records[0] } },
  async onLoad() { if (!guard()) return; this.name = getProfile().nickname; this.records = await listRecords() },
  methods: { ft: fmtTime }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: #F5F7F6; padding-top: 16rpx; }
.bn { display: flex; align-items: center; gap: 12rpx; margin: 0 32rpx 24rpx; padding: 16rpx 24rpx; border-radius: 20rpx; background: #FFF5E0; color: #A06C00; } .bn image { width: 32rpx; height: 32rpx; flex: none; }
.bp { align-items: baseline; margin-top: 8rpx; } .v { font-size: 80rpx; color: #000; line-height: 1.1; } .u { margin-left: 8rpx; } .hr { margin-left: auto; }
</style>
