<template>
  <view>
    <view class="seg"><view v-for="(s, i) in SEGS" :key="s.t" :class="{ on: seg === i }" @tap="seg = i">{{ s.t }}</view></view>
    <view class="card ch">
      <view class="top"><text class="f16 c0 b">血压变化</text><view class="lg f10 c6"><text><text class="dot" style="background:#10C495"></text>收缩压</text><text><text class="dot" style="background:#37A5E6"></text>舒张压</text></view></view>
      <view class="f12 c9 rg">{{ rangeStr }} · 单位 mmHg{{ chart.pts.length ? ' · 按日平均' : '' }}</view>
      <image v-if="chart.src" class="chart" :src="chart.src" mode="widthFix" />
      <view v-else class="empty">这段时间还没有记录</view>
    </view>
    <view class="avg">
      <view class="card"><view class="k f12 c6"><image src="/static/icons/sun-o.svg" />早间平均</view><view class="v num">{{ sa.morning }}<text class="sm">mmHg</text></view><view class="f10 c9">{{ sa.mn }} 次 · 12:00 前</view></view>
      <view class="card"><view class="k f12 c6"><image src="/static/icons/moon-b.svg" />晚间平均</view><view class="v num">{{ sa.evening }}<text class="sm">mmHg</text></view><view class="f10 c9">{{ sa.en }} 次 · 12:00 后</view></view>
    </view>
    <view class="sh"><text class="t">历史记录</text><text class="f12 c9">共 {{ list.length }} 次</text></view>
    <view class="card lst">
      <view v-if="!list.length" class="empty">暂无记录</view>
      <view class="it" v-for="r in shown" :key="r.id" @tap="$emit('pick', r)">
        <view class="d"><view class="f14 c0 b">{{ dl(r.time) }}</view><view class="f10 c9 num">{{ hm(r.time) }}</view></view>
        <view class="vv flex1"><text class="num f20 c0">{{ r.sys }}/{{ r.dia }}</text><text class="sm">mmHg</text><text class="tg" v-for="t in (r.tags || [])" :key="t">{{ t }}</text></view>
        <text class="f12 c6">{{ r.hr ? '心率 ' + r.hr : '' }}</text>
        <image v-if="!readonly" class="arrow ar" src="/static/icons/right-9.svg" />
      </view>
      <view v-if="list.length > shown.length" class="more f12 cg" @tap="limit += 30">查看更多</view>
    </view>
  </view>
</template>
<script>
import { buildChart, sessionAvg } from '../../common/chart.js'
import { dayLabel, hhmm } from '../../api.js'
export default {
  props: { records: { type: Array, default: () => [] }, readonly: Boolean },
  emits: ['pick'],
  data: () => ({ seg: 0, limit: 30, SEGS: [{ t: '近7天', d: 7 }, { t: '近30天', d: 30 }, { t: '全部', d: 0 }] }),
  computed: {
    days() { return this.SEGS[this.seg].d },
    list() { const d = this.days; if (!d) return this.records; const n = new Date(); const s = new Date(n.getFullYear(), n.getMonth(), n.getDate() - d + 1).getTime(); return this.records.filter(r => r.time >= s) },
    shown() { return this.list.slice(0, this.limit) },
    chart() { return buildChart(this.list, this.days) },
    sa() { return sessionAvg(this.list) },
    rangeStr() { const p = this.chart.pts; const f = t => { const d = new Date(t); return `${d.getMonth() + 1}月${d.getDate()}日` }
      if (this.days) { const n = new Date(); return f(new Date(n.getFullYear(), n.getMonth(), n.getDate() - this.days + 1)) + ' – ' + f(n) }
      return p.length ? f(p[0].t) + ' – ' + f(p[p.length - 1].t) : '全部' }
  },
  methods: { dl: dayLabel, hm: hhmm }
}
</script>
<style lang="scss" scoped>
.seg { margin: 8rpx 32rpx 24rpx; background: #EAEEEC; border-radius: 20rpx; padding: 6rpx; display: flex; }
.seg view { flex: 1; height: 60rpx; border-radius: 16rpx; display: flex; align-items: center; justify-content: center; font-size: 28rpx; color: #666; }
.seg .on { background: #fff; color: #000; font-weight: 600; box-shadow: 0 2rpx 8rpx rgba(0,0,0,.08); }
.top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12rpx; }
.lg { display: flex; gap: 24rpx; } .dot { display: inline-block; width: 16rpx; height: 16rpx; border-radius: 8rpx; margin-right: 8rpx; }
.rg { margin-bottom: 12rpx; }
.chart { width: 100%; display: block; }
.avg { display: flex; gap: 24rpx; margin: 0 32rpx 24rpx; }
.avg .card { margin: 0; flex: 1; padding: 28rpx; }
.k { display: flex; align-items: center; gap: 8rpx; } .k image { width: 28rpx; height: 28rpx; }
.avg .v { font-size: 40rpx; color: #000; margin: 12rpx 0 4rpx; }
.sm { font-size: 20rpx; color: #999; font-weight: 400; margin-left: 6rpx; font-family: "PingFang SC", sans-serif; }
.sh { margin-top: 8rpx; }
.lst { padding-top: 8rpx; padding-bottom: 8rpx; }
.it { display: flex; align-items: center; padding: 24rpx 0; } .it + .it { border-top: 1rpx solid #F0F2F1; }
.d { width: 140rpx; flex: none; }
.vv { display: flex; align-items: baseline; flex-wrap: wrap; }
.tg { font-size: 20rpx; color: #666; background: #F2F4F3; border-radius: 16rpx; padding: 2rpx 12rpx; margin-left: 10rpx; }
.ar { margin-left: 8rpx; flex: none; }
.more { text-align: center; padding: 20rpx 0 12rpx; border-top: 1rpx solid #F0F2F1; }
</style>
