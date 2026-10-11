<template>
  <view class="pg">
    <view class="safe-top"></view>
    <view class="navh f20 c0 b">记录</view>
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
<style scoped>.pg { min-height: 100vh; background: #F5F7F6; }</style>
