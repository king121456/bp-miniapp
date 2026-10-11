// 数据层：现阶段存本地，后端（Go + MongoDB）接口就绪后只改这个文件
const BASE = '' // TODO 后端地址
const USE_LOCAL = true
const K = { rec: 'bp_records', consent: 'health_consent', meds: 'meds', checkin: 'med_checkin' }
const get = k => uni.getStorageSync(k) || []
const req = (url, method = 'GET', data) => new Promise((ok, no) => uni.request({ url: BASE + url, method, data,
  header: { Authorization: 'Bearer ' + (uni.getStorageSync('token') || '') }, success: r => { const b = r.data || {}; if (b.needSubscribe) askSubscribe(); b.code === 0 ? ok(b.data) : no(b) }, fail: no }))
// 推送授权次数不够时，用大字弹窗先说明，用户同意后再调起微信授权
const TMPL = [] // TODO 填订阅消息模板 ID
function askSubscribe() {
  uni.showModal({ title: '继续提醒你', content: '要不要明天继续按时提醒你测量和用药？', confirmText: '好', cancelText: '不用了',
    success: r => r.confirm && TMPL.length && uni.requestSubscribeMessage({ tmplIds: TMPL }) })
}
const today = () => new Date().toDateString()

export const hasConsent = () => !!uni.getStorageSync(K.consent)
export async function login() {
  const consent = { agreedAt: Date.now(), version: '1.0' }
  if (USE_LOCAL) return uni.setStorageSync(K.consent, consent)
  const { code } = await uni.login({ provider: 'weixin' })
  const r = await req('/api/auth/login', 'POST', { code, consent })
  uni.setStorageSync('token', r.token); uni.setStorageSync(K.consent, consent)
}
export async function listRecords() { return USE_LOCAL ? get(K.rec) : req('/api/bp') }
export async function addRecord(r) {
  if (!USE_LOCAL) return req('/api/bp', 'POST', r)
  const l = get(K.rec); l.unshift({ id: String(Date.now()), time: Date.now(), ...r }); uni.setStorageSync(K.rec, l); return {}
}
// 每种药按服药时间点拆成一行：{ id, name, dose, time: '08:00', done }
export async function listMeds() {
  if (!USE_LOCAL) { const meds = await req('/api/meds'); return meds.flatMap(m => (m.slots || []).map(s => ({ ...m, time: s.time, done: s.done }))) }
  const done = get(K.checkin).filter(c => new Date(c.at).toDateString() === today()).map(c => c.medId + '@' + c.time)
  return get(K.meds).flatMap(m => (m.times || []).map(t => ({ ...m, time: t, done: done.includes(m.id + '@' + t) })))
}
export async function checkin(medId, time) {
  if (!USE_LOCAL) return req('/api/meds/' + medId + '/checkin', 'POST', { time })
  const l = get(K.checkin); l.push({ medId, time, at: Date.now() }); uni.setStorageSync(K.checkin, l); return {}
}
export const fmtTime = t => { const d = new Date(t), p = n => String(n).padStart(2, '0'); return `${d.getMonth()+1}月${d.getDate()}日 ${p(d.getHours())}:${p(d.getMinutes())}` }
export const hhmm = t => fmtTime(t).split(' ')[1]
