<template>
  <view class="pg">
    <view class="sec f12 c9">血压读数</view>
    <view class="card form">
      <view v-for="(c, i) in cells" :key="c.k" class="row" :class="{ on: focus === i }">
        <view class="lab"><view class="f16 c0 b">{{ c.a }}</view><view class="f10 c9">{{ c.b }}</view></view>
        <input class="ipt num" type="number" maxlength="3" :value="vals[i]" :focus="focus === i" :placeholder="c.ph" placeholder-class="phn"
          :confirm-type="i < 2 ? 'next' : 'done'" @input="onIn(i, $event)" @focus="focus = i" @blur="focus === i && (focus = -1)" @confirm="next(i)" />
        <view class="unit f12 c9">{{ c.u }}</view>
      </view>
    </view>
    <view class="sec f12 c9">测量时间</view>
    <view class="card form">
      <view class="row">
        <view class="lab"><view class="f14 c3">日期</view></view>
        <picker mode="date" :value="dStr" :end="today" @change="setD"><view class="pk f14 c0">{{ dStr }}<image src="/static/icons/right-9.svg" /></view></picker>
      </view>
      <view class="row">
        <view class="lab"><view class="f14 c3">时间</view></view>
        <picker mode="time" :value="tStr" @change="setT"><view class="pk f14 c0">{{ tStr }}<image src="/static/icons/right-9.svg" /></view></picker>
      </view>
    </view>
    <view class="sec f12 c9">身体感受 · 可多选，可不选</view>
    <view class="card blk">
      <view class="tags"><view v-for="t in TAGS" :key="t" class="tag" :class="{ on: tags.includes(t) }" @tap="toggle(t)">{{ t }}</view></view>
      <textarea class="note f14" v-model="note" placeholder="备注，比如刚运动完、刚起床" placeholder-class="phc" maxlength="60" auto-height />
    </view>
    <view v-if="editId" class="del f14 c6" @tap="del">删除这条记录</view>
    <view style="height: 220rpx"></view>
    <view class="bar">
      <view class="err f12 c6" v-if="err">{{ err }}</view>
      <view class="btn" :class="{ dis: !ready }" @tap="save">{{ editId ? '保存修改' : '保存记录' }}</view>
    </view>
  </view>
</template>
<script>
import { addRecord, updateRecord, deleteRecord, getRecord, validate, fmtTime, guard } from '../../api.js'
export default {
  data: () => ({ vals: ['', '', ''], focus: 0, tags: [], note: '', err: '', editId: '', time: Date.now(),
    TAGS: ['头晕', '头痛', '心慌', '乏力', '其他'],
    cells: [{ k: 's', a: '高压', b: '收缩压', u: 'mmHg', ph: '例如 120' }, { k: 'd', a: '低压', b: '舒张压', u: 'mmHg', ph: '例如 80' }, { k: 'h', a: '心率', b: '选填', u: '次/分', ph: '可不填' }] }),
  computed: {
    dStr() { const d = new Date(this.time); return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0') },
    tStr() { const d = new Date(this.time); return String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0') },
    today() { const d = new Date(); return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0') },
    ready() { return this.vals[0] && this.vals[1] }
  },
  async onLoad(q) {
    if (!guard()) return
    if (q && q.id) {
      const r = await getRecord(q.id); if (!r) return
      this.editId = r.id; this.time = r.time; this.vals = [String(r.sys), String(r.dia), r.hr ? String(r.hr) : '']; this.tags = r.tags || []; this.note = r.note || ''; this.focus = -1
      uni.setNavigationBarTitle({ title: '编辑记录' })
    }
  },
  methods: {
    onIn(i, e) { const v = String(e.detail.value).replace(/\D/g, '').slice(0, 3); this.vals.splice(i, 1, v); this.err = ''; if (v.length === 3 && i < 2) this.focus = i + 1; return v },
    next(i) { this.focus = i < 2 ? i + 1 : -1 },
    setD(e) { const [y, m, d] = e.detail.value.split('-').map(Number); const t = new Date(this.time); t.setFullYear(y, m - 1, d); this.time = Math.min(t.getTime(), Date.now()) },
    setT(e) { const [h, mi] = e.detail.value.split(':').map(Number); const t = new Date(this.time); t.setHours(h, mi); this.time = Math.min(t.getTime(), Date.now()) },
    toggle(t) { const i = this.tags.indexOf(t); i >= 0 ? this.tags.splice(i, 1) : this.tags.push(t) },
    async save() {
      if (!this.ready) { this.err = '请先填写高压和低压'; return }
      const r = { time: this.time, sys: +this.vals[0] || 0, dia: +this.vals[1] || 0, hr: +this.vals[2] || 0, tags: [...this.tags], note: this.note.trim() }
      const e = validate(r); if (e) { this.err = e; return }
      if (this.editId) await updateRecord(this.editId, r); else await addRecord(r)
      uni.showToast({ title: '已保存', icon: 'success' })
      setTimeout(() => uni.navigateBack({ fail: () => uni.switchTab({ url: '/pages/index/index' }) }), 700)
    },
    del() {
      uni.showModal({ title: '删除记录', content: '删除后无法恢复，确定删除这条记录吗？', confirmColor: '#10C495', success: async r => {
        if (!r.confirm) return; await deleteRecord(this.editId); uni.showToast({ title: '已删除', icon: 'none' }); setTimeout(() => uni.navigateBack(), 600) } })
    }
  }
}
</script>
<style lang="scss" scoped>
.pg { min-height: 100vh; background: #F5F7F6; padding-top: 8rpx; }
.sec { margin: 32rpx 48rpx 16rpx; }
.form { padding: 0 32rpx; border-radius: 32rpx; }
.row { display: flex; align-items: center; min-height: 120rpx; position: relative; }
.row + .row { border-top: 1rpx solid #F0F2F1; }
.row.on::before { content: ""; position: absolute; left: -32rpx; top: 28rpx; bottom: 28rpx; width: 6rpx; border-radius: 0 4rpx 4rpx 0; background: linear-gradient(180deg, #34C9A3, #62E17D); }
.lab { width: 160rpx; }
.ipt { flex: 1; height: 100rpx; text-align: right; font-size: 64rpx; color: #000; }
.row.on .ipt { color: #10C495; }
.unit { width: 96rpx; text-align: right; }
.pk { display: flex; align-items: center; gap: 8rpx; } .pk image { width: 28rpx; height: 28rpx; }
.row picker { flex: 1; display: flex; justify-content: flex-end; }
.blk { padding: 28rpx 32rpx 24rpx; border-radius: 32rpx; }
.tags { display: flex; flex-wrap: wrap; gap: 16rpx; }
.tags .tag { margin: 0; }
.note { width: 100%; margin-top: 24rpx; min-height: 80rpx; border-radius: 20rpx; background: #F5F7F6; padding: 20rpx 24rpx; box-sizing: border-box; color: #333; }
.del { text-align: center; margin-top: 40rpx; }
.bar { position: fixed; left: 0; right: 0; bottom: 0; padding: 20rpx 32rpx calc(24rpx + env(safe-area-inset-bottom)); background: rgba(245,247,246,.94); backdrop-filter: blur(20px); z-index: 10; }
.err { text-align: center; margin-bottom: 12rpx; }
.btn.dis { opacity: .45; box-shadow: none; }
</style>
<style>.phn { color: #D6DBD9; font-size: 32rpx; font-family: "PingFang SC", sans-serif; font-weight: 400; }</style>
<style>.phc { color: #999; }</style>
