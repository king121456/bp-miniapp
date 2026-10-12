<template>
  <view class="pg">
    <view class="card intro">
      <view class="f16 c0 b">最近 7 天未打卡、未服的时段</view>
      <view class="f12 c6 d">如果那次药其实已经吃过、只是忘了点，可以补记上。补记的时段会标「补记」，7 天内可以撤销。</view>
    </view>
    <view class="card lst">
      <view v-if="!todo.length" class="empty">最近 7 天没有需要补记的时段</view>
      <view class="cell" v-for="s in todo" :key="s.medId + s.date + s.time">
        <view class="ic bg-k"><image src="/static/icons/pill-6.svg" /></view>
        <view class="flex1 mn"><view class="f14 c0 b">{{ s.name }}</view><view class="f12 c9">{{ md(s.date) }} <text class="num">{{ s.time }}</text> · {{ s.dose }}<text v-if="s.status === 'skipped'" class="sk">未服</text></view></view>
        <view class="mkb" @tap="pick = s">补记</view>
      </view>
    </view>
    <view class="sh" v-if="done.length"><text class="t">已补记</text><text class="f12 c9">点开可查看或撤销</text></view>
    <view class="card lst" v-if="done.length">
      <view class="cell" v-for="s in done" :key="s.medId + s.date + s.time" @tap="pick = s">
        <view class="ic bg-g"><image src="/static/icons/pill-g.svg" /></view>
        <view class="flex1 mn"><view class="f14 c0 b">{{ s.name }}<text class="mk">补记</text></view><view class="f12 c9">计划 {{ md(s.date) }} <text class="num">{{ s.time }}</text> · 补记于 {{ ft(s.recordedAt) }}</view></view>
        <image class="arrow" src="/static/icons/right-9.svg" />
      </view>
    </view>
    <view class="f10 c9 ft">补记只是补上记录，不代表建议补服药物。</view>
    <slot-sheet :item="pick" @close="pick = null" @done="load" />
  </view>
</template>
<script>
import { guard, listSlots, fmtTime } from '../../api.js'
export default {
  data: () => ({ slots: [], pick: null }),
  computed: { todo() { return this.slots.filter(s => s.canMakeup) }, done() { return this.slots.filter(s => s.makeup) } },
  onShow() { if (guard()) this.load() },
  methods: {
    async load() { this.pick = null; this.slots = await listSlots(7) },
    md(d) { const [, m, dd] = d.split('-').map(Number); return `${m}月${dd}日` }, ft: fmtTime
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: #F5F7F6; padding: 24rpx 0 48rpx; }
.d { margin-top: 8rpx; line-height: 1.6; }
.lst { padding-top: 8rpx; padding-bottom: 8rpx; }
.mn { margin-left: 24rpx; }
.mkb { height: 64rpx; padding: 0 32rpx; border-radius: 32rpx; font-size: 28rpx; font-weight: 600; color: #10C495; background: #E3F8F1; display: flex; align-items: center; }
.mk { margin-left: 12rpx; background: #F2F4F3; color: #666; padding: 2rpx 12rpx; border-radius: 12rpx; font-size: 20rpx; font-weight: 400; }
.ft { text-align: center; margin-top: 24rpx; }
.sk { margin-left: 12rpx; background: #EEF0EF; color: #888; padding: 2rpx 12rpx; border-radius: 12rpx; font-size: 20rpx; }
</style>
