<template>
  <view class="pg">
    <view class="safe-top"></view>
    <view class="navh f20 c0 b">记录</view>
    <view class="card rp" @tap="goReport"><view class="ic sm bg-g"><image src="/static/icons/chart-g.svg" /></view><view class="flex1 lt"><view class="f14 c0 b">就诊报告卡</view><view class="f10 c9">近7/30/90天汇总，保存图片给医生看</view></view><image class="arrow" src="/static/icons/right-9.svg" /></view>
    <bp-overview :records="records" @pick="pick" />
    <view style="height:40rpx"></view>
  </view>
</template>
<script>
import { guard, listRecords, deleteRecord } from '../../api.js'
export default {
  data: () => ({ records: [] }),
  async onShow() { if (guard()) this.records = await listRecords() },
  methods: {
    goReport() { uni.navigateTo({ url: '/pages/report/report' }) },
    pick(r) {
      uni.showActionSheet({ itemList: ['编辑', '删除'], success: ({ tapIndex }) => {
        if (tapIndex === 0) return uni.navigateTo({ url: '/pages/entry/entry?id=' + r.id })
        uni.showModal({ title: '删除记录', content: '删除后无法恢复，确定删除这条记录吗？', confirmColor: '#10C495', success: async m => {
          if (m.confirm) { await deleteRecord(r.id); this.records = await listRecords(); uni.showToast({ title: '已删除', icon: 'none' }) } } })
      } })
    }
  }
}
</script>
<style scoped>.pg { min-height: 100vh; background: #F5F7F6; } .rp { display: flex; align-items: center; padding: 24rpx 32rpx; } .ic.sm { width: 64rpx; height: 64rpx; border-radius: 20rpx; } .ic.sm image { width: 36rpx; height: 36rpx; } .lt { margin-left: 24rpx; } .flex1 { flex: 1; }</style>
