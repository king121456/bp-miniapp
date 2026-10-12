<template>
  <view class="pg">
    <view class="inv">
      <view class="f14 b">邀请家人查看我的记录</view>
      <view class="f12 op">把邀请码发给家人，家人在小程序中输入后即可只读查看你的血压记录</view>
      <view v-if="inv" class="code num">{{ inv.code }}</view>
      <view v-if="inv" class="f10 op">24 小时内有效，使用一次后失效</view>
      <view class="ib" @tap="gen">{{ inv ? '重新生成' : '生成 6 位邀请码' }}</view>
      <view v-if="inv" class="ib2 f12" @tap="copy">复制邀请码</view>
    </view>
    <view class="sh"><text class="t">正在关注我的家人</text><text class="f12 c9">{{ list.length }} 位</text></view>
    <view class="card lst">
      <view v-if="!list.length" class="empty">还没有家人绑定</view>
      <view class="cell" v-for="f in list" :key="f.id">
        <view class="av">{{ f.name.slice(-1) }}</view>
        <view class="flex1 mn"><view class="f14 c0 b">{{ f.name }}<text class="f12 c9 rel">{{ f.relation }}</text></view><view class="f10 c9">{{ since(f.boundAt) }} 绑定 · 只读查看</view></view>
        <view class="ub f12 c6" @tap="unbind(f)">解除绑定</view>
      </view>
    </view>
    <view class="card pv" @tap="view">
      <view class="ic bg-o"><image src="/static/icons/eye-o.svg" /></view>
      <view class="flex1 mn"><view class="f14 c0 b">预览家人看到的页面</view><view class="f12 c9">只读，可查看记录和变化图表</view></view>
      <image class="arrow" src="/static/icons/right-9.svg" />
    </view>
  </view>
</template>
<script>
import { guard, createInvite, getInvite, listFamily, unbindFamily } from '../../api.js'
export default {
  data: () => ({ inv: null, list: [] }),
  async onShow() { if (!guard()) return; this.inv = getInvite(); this.list = await listFamily() },
  methods: {
    async gen() { this.inv = await createInvite() },
    copy() { uni.setClipboardData({ data: this.inv.code, success: () => uni.showToast({ title: '已复制', icon: 'none' }) }) },
    since(t) { const d = new Date(t); return `${d.getMonth() + 1}月${d.getDate()}日` },
    unbind(f) { uni.showModal({ title: '解除绑定', content: `解除后，${f.name}将不能再查看你的记录。`, confirmColor: '#10C495', success: async r => { if (r.confirm) { await unbindFamily(f.id); this.list = await listFamily() } } }) },
    view() { uni.navigateTo({ url: '/pages/family/view' }) }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: #F5F7F6; padding-top: 24rpx; }
.inv { position: relative; margin: 0 32rpx 40rpx; border-radius: 40rpx; padding: 40rpx 36rpx; color: #fff; text-align: center; background: linear-gradient(120deg, #10C495 0%, #34C9A3 45%, #62E17D 100%); box-shadow: 0 24rpx 56rpx -16rpx rgba(16,196,149,.55); overflow: hidden; }
.op { opacity: .88; margin-top: 8rpx; }
.code { font-size: 88rpx; letter-spacing: 16rpx; margin-top: 24rpx; line-height: 1.1; }
.ib { margin: 28rpx auto 0; width: 360rpx; height: 80rpx; line-height: 80rpx; border-radius: 40rpx; background: #fff; color: #10C495; font-size: 28rpx; font-weight: 600; }
.ib2 { margin-top: 20rpx; opacity: .92; text-decoration: underline; }
.lst { padding-top: 8rpx; padding-bottom: 8rpx; }
.av { width: 80rpx; height: 80rpx; border-radius: 40rpx; background: #FFF5E0; color: #FBA900; font-size: 32rpx; font-weight: 600; display: flex; align-items: center; justify-content: center; }
.mn { margin-left: 24rpx; } .rel { margin-left: 12rpx; font-weight: 400; }
.ub { padding: 10rpx 20rpx; border-radius: 28rpx; background: #F2F4F3; }
.pv { display: flex; align-items: center; }
</style>
