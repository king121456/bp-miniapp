<template>
  <view class="pg" v-if="r">
    <view class="card">
      <view class="cell" v-for="s in SL" :key="s.k">
        <view class="ic" :class="s.bg || ''"><image :src="'/static/icons/' + s.i + '.svg'" /></view>
        <view class="flex1 mn"><view class="f14 c0 b">{{ s.t }}</view>
          <picker mode="time" :value="r[s.k].time" @change="e => upd(s.k, 'time', e.detail.value)"><view class="row"><text class="f20 num" :class="r[s.k].on ? 'c0' : 'c9'">{{ r[s.k].time }}</text><text class="f12 c9 ed">点击修改</text></view></picker>
        </view>
        <switch :checked="r[s.k].on" color="#10C495" @change="e => upd(s.k, 'on', e.detail.value)" />
      </view>
    </view>
    <view class="note f12 c6">
      <view class="b c3 f14 nt">关于提醒</view>
      <view>· 设置会同步到服务器，到点时提醒你测量血压。</view>
      <view>· 正式版需要你在微信中允许「订阅消息」后，才能收到微信服务通知推送；每次允许对应一次推送。</view>
    </view>
  </view>
</template>
<script>
import { guard, getRemind, saveRemind } from '../../api.js'
export default {
  data: () => ({ r: null, SL: [{ k: 'morning', t: '早间测量', i: 'sun-o', bg: 'bg-o' }, { k: 'evening', t: '晚间测量', i: 'moon-b', bg: 'bg-b' }] }),
  async onLoad() { if (guard()) this.r = JSON.parse(JSON.stringify(await getRemind())) },
  methods: { async upd(k, f, v) { this.r[k][f] = v; await saveRemind(this.r); uni.showToast({ title: '已保存', icon: 'none' }) } }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: #F5F7F6; padding-top: 24rpx; }
.mn { margin-left: 24rpx; } .ed { margin-left: 16rpx; }
.note { margin: 16rpx 48rpx; line-height: 1.8; } .nt { margin-bottom: 8rpx; }
</style>
