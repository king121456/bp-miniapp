<template>
  <view class="page">
    <view class="time muted">{{ nowText }} · 自动记录</view>
    <view class="pair">
      <view :class="['f', cur === 'sys' && 'on']" @tap="cur = 'sys'"><view class="l"><text class="lb">高压</text>收缩压 mmHg</view><view :class="['v num', !form.sys && 'ph']">{{ form.sys || '--' }}</view></view>
      <view :class="['f', cur === 'dia' && 'on']" @tap="cur = 'dia'"><view class="l"><text class="lb">低压</text>舒张压 mmHg</view><view :class="['v num', !form.dia && 'ph']">{{ form.dia || '--' }}</view></view>
    </view>
    <view :class="['f hr', cur === 'hr' && 'on']" @tap="cur = 'hr'"><view class="l"><text class="lb">心率</text>次/分 · 可不填</view><view :class="['v2 num', !form.hr && 'ph']">{{ form.hr || '--' }}</view></view>
    <view class="q">有没有不舒服？<text class="muted aux">可不选</text></view>
    <view class="tags"><view v-for="t in tags" :key="t" :class="['tag', form.tags.includes(t) && 'on']" @tap="toggle(t)">{{ t }}</view></view>
    <view class="note-wrap"><input class="note" v-model="form.note" placeholder="备注（可不填）" maxlength="100" /></view>
    <view class="save"><view class="btn-primary" @tap="save">保存这次记录</view></view>
    <view class="kb"><view v-for="k in keys" :key="k" class="k" :class="!k && 'blank'" @tap="press(k)">{{ k }}</view></view>
  </view>
</template>
<script setup>
import { reactive, ref } from 'vue'
import { addRecord, hasConsent, fmtTime } from '../../api.js'
const tags = ['头晕', '头痛', '心慌', '乏力']
const keys = ['1','2','3','4','5','6','7','8','9','','0','⌫']
const form = reactive({ sys: '', dia: '', hr: '', tags: [], note: '' }), cur = ref('sys')
const nowText = fmtTime(Date.now())
const press = k => {
  if (!k) return
  const v = form[cur.value]
  if (k === '⌫') return (form[cur.value] = v.slice(0, -1))
  if (v.length >= 3) return
  form[cur.value] = v + k
  // 填满 3 位自动跳到下一项
  if (form[cur.value].length === 3) cur.value = { sys: 'dia', dia: 'hr', hr: 'hr' }[cur.value]
}
const toggle = t => { const i = form.tags.indexOf(t); i > -1 ? form.tags.splice(i, 1) : form.tags.push(t) }
const save = async () => {
  if (!hasConsent()) return uni.reLaunch({ url: '/pages/login/login' })
  const s = +form.sys, d = +form.dia, h = +form.hr
  if (!s || !d) return uni.showToast({ title: '请填写高压和低压', icon: 'none', duration: 2500 })
  if (s < 40 || s > 300 || d < 20 || d > 200 || (h && (h < 20 || h > 250))) return uni.showToast({ title: '数字好像输错了，请再看看', icon: 'none', duration: 2500 })
  await addRecord({ sys: s, dia: d, hr: h || null, tags: [...form.tags], note: form.note })
  uni.showToast({ title: '已保存', duration: 2000 }); setTimeout(() => uni.navigateBack(), 1200)
}
</script>
<style lang="scss">
.page{min-height:100vh;background:#fff;padding-bottom:460rpx}
.time{display:inline-block;margin:12rpx 40rpx 36rpx;background:$c-bg;border-radius:32rpx;padding:12rpx 28rpx;font-size:$fs-aux}
.pair{display:flex;gap:24rpx;margin:0 $gap 24rpx}
.f{flex:1;background:$c-input;border-radius:$radius;padding:32rpx;border:4rpx solid transparent}
.f.on{border-color:$c-primary-l;background:#fff;box-shadow:0 12rpx 36rpx rgba(16,196,149,.15)}
.l{font-size:34rpx;color:$c-text-2}.lb{display:block;color:$c-text;font-size:38rpx;font-weight:600}
.v{font-size:$fs-xl;line-height:1.15;margin-top:8rpx}.ph{color:$c-text-3}
.hr{margin:0 $gap 32rpx;display:flex;align-items:center;justify-content:space-between}.v2{font-size:64rpx}
.q{margin:0 40rpx 20rpx;font-size:38rpx;font-weight:600}.aux{font-size:$fs-aux;font-weight:400;margin-left:12rpx}
.tags{display:flex;gap:16rpx;margin:0 $gap 24rpx}
.tag{height:$hit;padding:0 40rpx;border-radius:44rpx;background:$c-bg;display:flex;align-items:center;white-space:nowrap}
.tag.on{background:$c-primary-bg;color:$c-primary;font-weight:600;box-shadow:inset 0 0 0 4rpx $c-primary-l}
.note-wrap{margin:0 $gap}.note{height:$hit;background:$c-bg;border-radius:24rpx;padding:0 28rpx;font-size:$fs-body}
.save{position:fixed;left:$gap;right:$gap;bottom:470rpx}
.kb{position:fixed;left:0;right:0;bottom:0;background:$c-line;padding:16rpx 16rpx calc(16rpx + env(safe-area-inset-bottom));display:grid;grid-template-columns:repeat(3,1fr);gap:16rpx}
.k{height:92rpx;background:#fff;border-radius:20rpx;display:flex;align-items:center;justify-content:center;font-size:52rpx}.k.blank{background:transparent}
</style>
