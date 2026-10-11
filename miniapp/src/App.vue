<script>
export default { onLaunch() {} }
</script>
<style lang="scss">
page { background: #F5F7F6; color: #333; font-size: 28rpx; line-height: 1.5; font-family: "PingFang SC", "Noto Sans CJK SC", "Helvetica Neue", sans-serif; -webkit-font-smoothing: antialiased; }
view, text, input, textarea, button { box-sizing: border-box; }
.num { font-family: "DIN Alternate", "D-DIN-EXP", "DIN Condensed", "Helvetica Neue", Arial, sans-serif; font-weight: 700; letter-spacing: -0.6rpx; }
.f10 { font-size: 20rpx; } .f12 { font-size: 24rpx; } .f14 { font-size: 28rpx; } .f16 { font-size: 32rpx; } .f20 { font-size: 40rpx; }
.c0 { color: #000; } .c3 { color: #333; } .c6 { color: #666; } .c9 { color: #999; } .cg { color: #10C495; }
.b { font-weight: 600; }
.card { background: #fff; border-radius: 32rpx; margin: 0 32rpx 24rpx; padding: 32rpx; box-shadow: 0 2rpx 4rpx rgba(16,60,48,.04), 0 12rpx 40rpx rgba(16,60,48,.05); }
.row { display: flex; align-items: center; }
.flex1 { flex: 1; min-width: 0; }
.ic { width: 80rpx; height: 80rpx; border-radius: 24rpx; display: flex; align-items: center; justify-content: center; flex: none; }
.ic image { width: 44rpx; height: 44rpx; }
.bg-g { background: #E3F8F1; } .bg-o { background: #FFF5E0; } .bg-b { background: #E6F4FC; } .bg-r { background: #FDECEC; } .bg-k { background: #F2F4F3; }
.btn { height: 96rpx; border-radius: 48rpx; background: linear-gradient(120deg, #10C495, #34C9A3 50%, #62E17D); color: #fff; font-size: 32rpx; font-weight: 600; display: flex; align-items: center; justify-content: center; box-shadow: 0 20rpx 48rpx -16rpx rgba(16,196,149,.6); }
.btn.dis { opacity: .4; box-shadow: none; }
.btn.ghost { background: #fff; color: #10C495; box-shadow: 0 2rpx 4rpx rgba(16,60,48,.04), 0 12rpx 40rpx rgba(16,60,48,.06); }
.btn.mini { height: 60rpx; padding: 0 28rpx; border-radius: 30rpx; font-size: 24rpx; box-shadow: 0 8rpx 20rpx -4rpx rgba(16,196,149,.5); }
.sh { display: flex; justify-content: space-between; align-items: center; margin: 0 40rpx 20rpx; }
.sh .t { font-size: 32rpx; color: #000; font-weight: 600; }
.cell { display: flex; align-items: center; padding: 24rpx 0; }
.cell + .cell { border-top: 1rpx solid #F0F2F1; }
.arrow { width: 28rpx; height: 28rpx; }
.empty { text-align: center; padding: 48rpx 0; color: #999; font-size: 24rpx; }
.navh { padding: 16rpx 40rpx 24rpx; }
.safe-top { height: var(--status-bar-height); }
/* #ifdef MP-WEIXIN */
.safe-top { height: 88rpx; }
/* #endif */
.tag { height: 60rpx; padding: 0 28rpx; border-radius: 30rpx; background: #F5F7F6; font-size: 24rpx; display: inline-flex; align-items: center; color: #333; margin: 0 16rpx 16rpx 0; }
.tag.on { background: #E3F8F1; color: #10C495; font-weight: 600; box-shadow: inset 0 0 0 2rpx #10C495; }
</style>
