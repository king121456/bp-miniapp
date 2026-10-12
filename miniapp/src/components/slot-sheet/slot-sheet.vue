<template>
  <view class="mask" v-if="item" @tap="close">
    <view class="sheet" @tap.stop>
      <!-- 补记确认（最多 2 步：点「补记」→ 确认） -->
      <block v-if="item.canMakeup">
        <view class="f20 c0 b ttl">补记</view>
        <view class="f16 c3 q">补记：{{ md }} <text class="num">{{ item.time }}</text> 的{{ item.name }}{{ item.dose ? ' ' + item.dose : '' }}，确认你那次已经服过了吗？</view>
        <picker mode="time" :value="taken || item.time" @change="e => taken = e.detail.value">
          <view class="cell pk"><text class="f14 c6 flex1">实际服药时间（选填）</text><text class="f14 num" :class="taken ? 'c0' : 'c9'">{{ taken || '不填' }}</text><image class="arrow" src="/static/icons/right-9.svg" /></view>
        </picker>
        <view class="f12 c9 nt">补记只是补上记录，不代表建议补服药物。</view>
        <view class="acts"><view class="btn ghost flex1" @tap="close">取消</view><view class="btn flex1" @tap="ok">确认补记</view></view>
      </block>
      <!-- 补记详情 / 撤销 -->
      <block v-else>
        <view class="f20 c0 b ttl">{{ item.name }}<text class="mk f12">补记</text></view>
        <view class="kv f14"><text class="c9">计划时间</text><text class="c0">{{ md }} <text class="num">{{ item.time }}</text></text></view>
        <view class="kv f14"><text class="c9">补记于</text><text class="c0">{{ ft(item.recordedAt) }}</text></view>
        <view class="kv f14" v-if="item.takenTime"><text class="c9">自填服药时间</text><text class="c0 num">{{ item.takenTime }}</text></view>
        <view class="acts" v-if="!readonly && item.canUndo"><view class="btn ghost flex1" @tap="close">关闭</view><view class="btn ghost flex1 un" @tap="undo">撤销补记</view></view>
        <view class="acts" v-else><view class="btn ghost flex1" @tap="close">关闭</view></view>
      </block>
    </view>
  </view>
</template>
<script>
import { makeupCheckin, undoMakeup, fmtTime } from '../../api.js'
export default {
  props: { item: Object, readonly: Boolean },
  emits: ['close', 'done'],
  data: () => ({ taken: '' }),
  watch: { item() { this.taken = '' } },
  computed: { md() { if (!this.item) return ''; const [, m, d] = this.item.date.split('-').map(Number); return `${m}月${d}日` } },
  methods: {
    ft: fmtTime,
    close() { this.$emit('close') },
    async ok() {
      try { await makeupCheckin(this.item, this.taken); uni.showToast({ title: '已补记', icon: 'none' }); this.$emit('done') }
      catch (e) { uni.showToast({ title: (e && e.message) || '补记失败', icon: 'none' }) }
    },
    undo() {
      uni.showModal({ title: '撤销补记', content: '撤销后，这个时段会恢复为未打卡。', confirmText: '撤销', confirmColor: '#10C495', success: async r => {
        if (!r.confirm) return
        await undoMakeup(this.item); uni.showToast({ title: '已撤销', icon: 'none' }); this.$emit('done')
      } })
    }
  }
}
</script>
<style lang="scss" scoped>
.mask { position: fixed; inset: 0; background: rgba(0,0,0,.45); z-index: 99; display: flex; align-items: flex-end; }
.sheet { width: 100%; background: #fff; border-radius: 32rpx 32rpx 0 0; padding: 40rpx 40rpx calc(40rpx + env(safe-area-inset-bottom)); }
.ttl { margin-bottom: 24rpx; display: flex; align-items: center; }
.q { line-height: 1.6; margin-bottom: 16rpx; }
.pk { background: #F5F7F6; border-radius: 24rpx; padding: 24rpx 28rpx; margin-bottom: 16rpx; }
.nt { margin-bottom: 32rpx; }
.kv { display: flex; justify-content: space-between; padding: 20rpx 0; border-bottom: 1rpx solid #F0F2F1; }
.acts { display: flex; gap: 24rpx; margin-top: 32rpx; }
.un { color: #666; }
.mk { margin-left: 16rpx; background: #F2F4F3; color: #666; padding: 2rpx 14rpx; border-radius: 12rpx; font-weight: 400; }
</style>
