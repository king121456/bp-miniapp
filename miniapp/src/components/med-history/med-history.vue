<template>
  <view>
    <view v-if="!days.length" class="card empty">暂无服药记录</view>
    <view class="card lst" v-for="d in days" :key="d.date">
      <view class="dh"><text class="f14 c0 b">{{ d.label }}</text><text class="f12 c9">已服 {{ d.taken }}/{{ d.total }}</text></view>
      <view class="cell" v-for="s in d.items" :key="s.medId + s.time" @tap="s.makeup && $emit('pick', s)">
        <text class="num f16 c0 tm">{{ s.time }}</text>
        <view class="flex1 mn"><view class="f14 c0">{{ s.name }}</view>
          <view class="f10 c9">{{ s.dose }}<text v-if="s.status === 'taken' && !s.makeup"> · {{ hm(s.checkedAt) }} 打卡</text><text v-if="s.makeup"> · 已于 {{ ft(s.recordedAt) }} 补记</text></view></view>
        <view v-if="s.status === 'taken'" class="st ok"><image src="/static/icons/check-l.svg" />已服<text v-if="s.makeup" class="mk">补记</text></view>
        <view v-else-if="s.canMakeup && !readonly" class="mkb" @tap.stop="$emit('pick', s)">补记</view>
        <text v-else class="f12 c9">{{ s.status === 'missed' ? '未打卡' : s.status === 'pending' ? '待服' : '未到时间' }}</text>
      </view>
    </view>
  </view>
</template>
<script>
import { hhmm, fmtTime, WEEK } from '../../api.js'
export default {
  props: { slots: { type: Array, default: () => [] }, readonly: Boolean },
  emits: ['pick'],
  computed: {
    days() {
      const g = {}, out = []
      for (const s of this.slots) { if (!g[s.date]) { g[s.date] = { date: s.date, items: [] }; out.push(g[s.date]) } g[s.date].items.push(s) }
      const td = new Date(); td.setHours(0, 0, 0, 0)
      return out.map(d => {
        const [y, m, dd] = d.date.split('-').map(Number), dt = new Date(y, m - 1, dd), diff = Math.round((td - dt) / 864e5)
        d.items.sort((a, b) => a.time.localeCompare(b.time))
        const due = d.items.filter(s => s.status !== 'future')
        return { ...d, label: (diff === 0 ? '今天' : diff === 1 ? '昨天' : `${m}月${dd}日`) + ' 星期' + WEEK[dt.getDay()], total: due.length, taken: due.filter(s => s.status === 'taken').length }
      })
    }
  },
  methods: { hm: hhmm, ft: fmtTime }
}
</script>
<style lang="scss" scoped>
.lst { padding-top: 24rpx; padding-bottom: 8rpx; }
.dh { display: flex; justify-content: space-between; align-items: center; padding-bottom: 12rpx; border-bottom: 1rpx solid #F0F2F1; }
.tm { width: 96rpx; } .mn { margin-left: 8rpx; }
.st { display: flex; align-items: center; gap: 6rpx; font-size: 24rpx; font-weight: 600; color: #31CD4C; } .st image { width: 24rpx; height: 24rpx; }
.mk { margin-left: 8rpx; background: #F2F4F3; color: #666; padding: 2rpx 12rpx; border-radius: 12rpx; font-size: 20rpx; font-weight: 400; }
.mkb { height: 64rpx; padding: 0 32rpx; border-radius: 32rpx; font-size: 28rpx; font-weight: 600; color: #10C495; background: #E3F8F1; display: flex; align-items: center; }
</style>
