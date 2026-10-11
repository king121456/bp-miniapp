<template>
  <view class="pg">
    <view class="time"><view class="tp"><image src="/static/icons/clock-6.svg" />{{ timeStr }} · {{ editId ? '编辑记录' : '自动记录' }}</view></view>
    <view class="big">
      <view v-for="(c, i) in cells" :key="c.k" class="bc" :class="{ on: active === i }" @tap="active = i">
        <view class="f12 c6"><text class="f14 c0 b kb2">{{ c.a }}</text>{{ c.b }}</view>
        <view class="v num" :class="{ ph: !vals[i] && active !== i }">{{ vals[i] || (active === i ? '' : '--') }}<text v-if="active === i" class="caret"></text></view>
        <view class="f10 c9">{{ c.u }}</view>
      </view>
    </view>
    <view class="card blk">
      <view class="f14 c0 b h">身体感受<text class="f12 c9 hs">可多选，可不选</text></view>
      <view class="tags"><view v-for="t in TAGS" :key="t" class="tag" :class="{ on: tags.includes(t) }" @tap="toggle(t)">{{ t }}</view></view>
      <input class="note f12" v-model="note" placeholder="添加备注…" placeholder-class="phc" maxlength="60" @focus="active = -1" />
    </view>
    <view class="err f12 c6" v-if="err">{{ err }}</view>
    <view class="acts">
      <view class="btn" @tap="save">保存</view>
      <view v-if="editId" class="del f14 c6" @tap="del">删除这条记录</view>
    </view>
    <view style="height: 520rpx"></view>
    <view class="kb" v-if="active >= 0">
      <view v-for="k in KEYS" :key="k" class="key num" :class="{ x: k === '' || k === 'del' }" @tap="press(k)">
        <image v-if="k === 'del'" src="/static/icons/del-k.svg" /><text v-else>{{ k }}</text>
      </view>
    </view>
  </view>
</template>
<script>
import { addRecord, updateRecord, deleteRecord, getRecord, validate, fmtTime, guard } from '../../api.js'
export default {
  data: () => ({ vals: ['', '', ''], active: 0, tags: [], note: '', err: '', editId: '', time: Date.now(),
    TAGS: ['头晕', '头痛', '心慌', '乏力', '其他'], KEYS: ['1', '2', '3', '4', '5', '6', '7', '8', '9', '', '0', 'del'],
    cells: [{ k: 's', a: '高压', b: '收缩压', u: 'mmHg' }, { k: 'd', a: '低压', b: '舒张压', u: 'mmHg' }, { k: 'h', a: '心率', b: '选填', u: '次/分' }] }),
  computed: { timeStr() { return fmtTime(this.time) } },
  async onLoad(q) {
    if (!guard()) return
    if (q && q.id) {
      const r = await getRecord(q.id); if (!r) return
      this.editId = r.id; this.time = r.time; this.vals = [String(r.sys), String(r.dia), r.hr ? String(r.hr) : '']; this.tags = r.tags || []; this.note = r.note || ''; this.active = -1
      uni.setNavigationBarTitle({ title: '编辑记录' })
    }
  },
  methods: {
    press(k) {
      if (k === '') return
      this.err = ''
      const i = this.active, v = this.vals[i]
      if (k === 'del') { if (v) this.vals.splice(i, 1, v.slice(0, -1)); else if (i > 0) this.active = i - 1; return }
      if (v.length >= 3) { if (i < 2) { this.active = i + 1; this.press(k) } return }
      const nv = v + k; this.vals.splice(i, 1, nv)
      if (nv.length === 3 && i < 2) this.active = i + 1
    },
    toggle(t) { const i = this.tags.indexOf(t); i >= 0 ? this.tags.splice(i, 1) : this.tags.push(t) },
    async save() {
      const r = { sys: +this.vals[0] || 0, dia: +this.vals[1] || 0, hr: +this.vals[2] || 0, tags: [...this.tags], note: this.note.trim() }
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
.time { display: flex; justify-content: center; margin: 8rpx 0 32rpx; }
.tp { display: inline-flex; align-items: center; gap: 8rpx; font-size: 24rpx; color: #666; background: #fff; padding: 10rpx 24rpx; border-radius: 28rpx; }
.tp image { width: 24rpx; height: 24rpx; }
.big { margin: 0 32rpx 24rpx; background: #fff; border-radius: 40rpx; padding: 40rpx 16rpx 36rpx; display: flex; box-shadow: 0 12rpx 40rpx rgba(16,60,48,.05); }
.bc { flex: 1; text-align: center; position: relative; }
.bc + .bc::before { content: ""; position: absolute; left: 0; top: 24rpx; bottom: 24rpx; width: 1rpx; background: #EEF1F0; }
.kb2 { margin-right: 8rpx; }
.v { font-size: 80rpx; line-height: 1.15; margin-top: 16rpx; color: #000; height: 92rpx; }
.v.ph { color: #D6DBD9; }
.on .v { color: #10C495; }
.on::after { content: ""; position: absolute; left: 50%; bottom: -36rpx; transform: translateX(-50%); width: 56rpx; height: 6rpx; border-radius: 4rpx; background: linear-gradient(90deg, #34C9A3, #62E17D); }
.caret { display: inline-block; width: 4rpx; height: 60rpx; background: #10C495; margin-left: 4rpx; vertical-align: -4rpx; border-radius: 2rpx; animation: bl 1s steps(1) infinite; }
@keyframes bl { 50% { opacity: 0; } }
.blk { padding: 28rpx 32rpx 32rpx; border-radius: 32rpx; }
.h { margin-bottom: 20rpx; } .hs { margin-left: 12rpx; font-weight: 400; }
.tags { display: flex; flex-wrap: wrap; }
.tags .tag { margin: 0 16rpx 0 0; }
.note { margin-top: 24rpx; height: 72rpx; border-radius: 20rpx; background: #F5F7F6; padding: 0 24rpx; color: #333; }
.err { margin: 0 48rpx 16rpx; text-align: center; }
.acts { padding: 8rpx 32rpx 0; }
.del { text-align: center; margin-top: 28rpx; }
.kb { position: fixed; bottom: 0; left: 0; right: 0; background: #D2D5DB; padding: 12rpx 12rpx calc(24rpx + env(safe-area-inset-bottom)); display: flex; flex-wrap: wrap; gap: 12rpx; z-index: 10; }
.key { width: calc((100% - 24rpx) / 3); height: 92rpx; background: #fff; border-radius: 12rpx; box-shadow: 0 2rpx 0 #898A8D; display: flex; align-items: center; justify-content: center; font-size: 40rpx; color: #000; }
.key:active { background: #E8EBEA; }
.key.x { background: transparent; box-shadow: none; }
.key image { width: 48rpx; height: 48rpx; }
</style>
<style>.phc { color: #999; }</style>
