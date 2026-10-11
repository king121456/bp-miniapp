<template>
  <view class="page">
    <view class="hello"><view class="h1">{{ greet }}</view><view class="muted">{{ dateStr }}</view></view>
    <view class="hero">
      <view class="row-b"><text>今日血压</text><text v-if="latest">{{ hhmm(latest.time) }} 测量</text></view>
      <view v-if="latest" class="v num">{{ latest.sys }}/{{ latest.dia }}<text class="u">mmHg</text></view>
      <view v-else class="empty">今天还没有记录</view>
      <view class="chips" v-if="latest">
        <view class="chip">心率 {{ latest.hr || '--' }} 次/分</view><view class="chip">今天已测 {{ todayCount }} 次</view>
      </view>
    </view>
    <view class="wrap"><view class="btn-primary" @tap="toEntry"><text class="plus">＋</text>记录血压</view></view>
    <view class="card">
      <view class="sec"><text class="h2">今日用药</text><text class="muted aux" v-if="meds.length">{{ doneCount }} / {{ meds.length }} 已服</text></view>
      <view v-if="!meds.length" class="muted">还没有添加药品</view>
      <view class="med" v-for="m in meds" :key="m.id + m.time">
        <view class="ic"><image src="/static/icons/pill.svg" class="ii" /></view>
        <view class="n"><view class="mn">{{ m.name }} {{ m.dose }}</view><view class="muted aux">{{ m.time }}{{ m.amount ? ' · ' + m.amount : '' }}</view></view>
        <view v-if="m.done" class="ck done">✓ 已服</view>
        <view v-else class="ck todo" @tap="doCheck(m)">打卡</view>
      </view>
    </view>
  </view>
</template>
<script setup>
import { ref, computed } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { hasConsent, listRecords, listMeds, checkin, hhmm } from '../../api.js'
const recs = ref([]), meds = ref([])
const isToday = t => new Date(t).toDateString() === new Date().toDateString()
const todayRecs = computed(() => recs.value.filter(r => isToday(r.time)))
const latest = computed(() => todayRecs.value[0]), todayCount = computed(() => todayRecs.value.length)
const doneCount = computed(() => meds.value.filter(m => m.done).length)
const h = new Date().getHours(), greet = h < 11 ? '早上好' : h < 13 ? '中午好' : h < 18 ? '下午好' : '晚上好'
const d = new Date(), dateStr = `${d.getMonth()+1}月${d.getDate()}日 星期${'日一二三四五六'[d.getDay()]}`
onShow(async () => {
  if (!hasConsent()) return uni.reLaunch({ url: '/pages/login/login' })
  recs.value = await listRecords(); meds.value = await listMeds()
})
const toEntry = () => uni.navigateTo({ url: '/pages/entry/entry' })
const doCheck = async m => { await checkin(m.id, m.time); m.done = true; uni.showToast({ title: '已打卡', duration: 2000 }) }
</script>
<style lang="scss">
.page{min-height:100vh;background:linear-gradient(180deg,$c-head 0,$c-bg 600rpx);padding-bottom:40rpx}
.hello{padding:20rpx 40rpx 36rpx}.h1{font-size:$fs-h1;font-weight:700}
.hero{margin:0 $gap 28rpx;padding:44rpx;border-radius:48rpx;color:#fff;background:linear-gradient(135deg,#10C495 0%,#34C9A3 55%,#62E17D 100%);box-shadow:0 28rpx 60rpx rgba(16,196,149,.3)}
.row-b{display:flex;justify-content:space-between;opacity:.85;font-size:34rpx}
.v{font-size:$fs-xxl;line-height:1.1;margin:20rpx 0 8rpx}.u{font-size:40rpx;font-weight:500;opacity:.8;margin-left:12rpx}
.empty{font-size:$fs-h1;font-weight:600;margin:28rpx 0 8rpx}
.chips{display:flex;gap:20rpx;margin-top:28rpx}.chip{background:rgba(255,255,255,.16);border-radius:28rpx;padding:16rpx 28rpx;font-size:34rpx}
.wrap{padding:0 $gap 28rpx}.plus{font-size:48rpx;margin-right:16rpx}
.sec{display:flex;justify-content:space-between;align-items:center;margin-bottom:24rpx}.h2{font-size:$fs-h2;font-weight:600}.aux{font-size:$fs-aux}
.med{display:flex;align-items:center;gap:28rpx;padding:24rpx 0;border-top:2rpx solid $c-line}
.ic{width:96rpx;height:96rpx;border-radius:28rpx;background:$c-primary-bg;display:flex;align-items:center;justify-content:center}.ii{width:52rpx;height:52rpx}
.n{flex:1}.mn{font-size:38rpx;font-weight:600}
.ck{height:$btn2-h;padding:0 36rpx;border-radius:44rpx;display:flex;align-items:center;font-size:34rpx;font-weight:600}
.ck.done{background:#F1F3F7;color:$c-text-2}.ck.todo{background:$c-primary;color:#fff}
</style>
