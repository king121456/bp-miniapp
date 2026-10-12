<template>
  <view class="mask" v-if="item" @tap="close">
    <view class="sheet" @tap.stop>
      <!-- 未服：可补记或撤销没吃 -->
      <block v-if="item.status === 'skipped' && !confirm">
        <view class="f20 c0 b ttl">{{ item.name }}<text class="sk f12">未服</text></view>
        <view class="kv f14"><text class="c9">计划时间</text><text class="c0">{{ md }} <text class="num">{{ item.time }}</text></text></view>
        <view class="kv f14" v-if="item.dose"><text class="c9">剂量</text><text class="c0">{{ item.dose }}</text></view>
        <view class="f12 c9 nt2" v-if="!readonly">这次记为「没吃」。如果其实吃过了，可以补记。</view>
        <view class="acts" v-if="!readonly && (item.canUnskip || item.canMakeup)"><view class="btn ghost flex1 un" v-if="item.canUnskip" @tap="unskip">撤销没吃</view><view class="btn flex1" v-if="item.canMakeup" @tap="confirm = true">补记</view></view>
        <view class="acts" v-else><view class="btn ghost flex1" @tap="close">关闭</view></view>
      </block>
      <!-- 补记确认（最多 2 步：点「补记」→ 确认） -->
      <block v-else-if="item.canMakeup && !readonly">
        <view class="f20 c0 b ttl">补记</view>
        <view class="f16 c3 q">补记：{{ md }} <text class="num">{{ item.time }}</text> 的{{ item.name }}{{ item.dose ? ' ' + item.dose : '' }}，确认你那次已经服过了吗？</view>
        <view class="f12 c9 nt">补记只是补上记录，不代表建议补服药物。</view>
        <view class="acts"><view class="btn ghost flex1" @tap="close">取消</view><view class="btn flex1" @tap="ok">确认补记</view></view>
      </block>
      <!-- 补记详情 / 撤销 -->
      <block v-else>
        <view class="f20 c0 b ttl">{{ item.name }}<text class="mk f12">补记</text></view>
        <view class="kv f14"><text class="c9">计划时间</text><text class="c0">{{ md }} <text class="num">{{ item.time }}</text></text></view>
        <view class="kv f14"><text class="c9">补记于</text><text class="c0">{{ ft(item.recordedAt) }}</text></view>
        <view class="acts" v-if="!readonly && item.canUndo"><view class="btn ghost flex1" @tap="close">关闭</view><view class="btn ghost flex1 un" @tap="undo">撤销补记</view></view>
        <view class="acts" v-else><view class="btn ghost flex1" @tap="close">关闭</view></view>
      </block>
    </view>
  </view>
</template>
<script>
import { makeupCheckin, undoMakeup, unskipSlot, fmtTime } from '../../api.js'
export default {
  props: { item: Object, readonly: Boolean },
  emits: ['close', 'done'],
  data: () => ({ confirm: false }),
  watch: {
    item(v) {
      this.confirm = false
      // #ifdef H5
      // H5 的 tabBar 会盖住底部弹层，打开时先隐藏（非 tab 页调用会静默失败）
      v ? uni.hideTabBar({ fail() {} }) : uni.showTabBar({ fail() {} })
      // #endif
    }
  },
  computed: { md() { if (!this.item) return ''; const [, m, d] = this.item.date.split('-').map(Number); return `${m}月${d}日` } },
  methods: {
    ft: fmtTime,
    close() { this.$emit('close') },
    async ok() {
      try { await makeupCheckin(this.item); uni.showToast({ title: '已补记', icon: 'none' }); this.$emit('done') }
      catch (e) { uni.showToast({ title: (e && e.message) || '补记失败', icon: 'none' }) }
    },
    async unskip() {
      try { await unskipSlot(this.item); uni.showToast({ title: '已撤销', icon: 'none' }); this.$emit('done') }
      catch (e) { uni.showToast({ title: (e && e.message) || '操作失败', icon: 'none' }) }
    },
    undo() {
      uni.showModal({ title: '撤销补记', content: '撤销后，这个时段会恢复为补记前的状态。', confirmText: '撤销', confirmColor: '#10C495', success: async r => {
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
.nt2 { margin-top: 24rpx; line-height: 1.6; }
.sk { margin-left: 16rpx; background: #EEF0EF; color: #888; padding: 2rpx 14rpx; border-radius: 12rpx; font-weight: 400; }
.nt { margin-bottom: 32rpx; }
.kv { display: flex; justify-content: space-between; padding: 20rpx 0; border-bottom: 1rpx solid #F0F2F1; }
.acts { display: flex; gap: 24rpx; margin-top: 32rpx; }
.un { color: #666; }
.mk { margin-left: 16rpx; background: #F2F4F3; color: #666; padding: 2rpx 14rpx; border-radius: 12rpx; font-weight: 400; }
</style>
