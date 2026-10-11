// 数据层：USE_LOCAL=true 时全部存本地（uni.setStorageSync），后端（Go + MongoDB，见 bp-server/README.md）就绪后把 USE_LOCAL 改为 false 并填 BASE
const BASE = '' // TODO 后端地址，如 https://api.example.com
export const USE_LOCAL = true
const K = { rec: 'bp_records', consent: 'health_consent', meds: 'meds', checkin: 'med_checkin', remind: 'measure_remind', family: 'family_members', invite: 'family_invite', profile: 'profile', seeded: 'demo_seeded' }
export const CONSENT_VERSION = '1.0'
const get = (k, d = []) => { const v = uni.getStorageSync(k); return v === '' || v == null ? d : v }
const set = (k, v) => uni.setStorageSync(k, v)
const uid = () => Date.now().toString(36) + Math.random().toString(36).slice(2, 6)
const req = (url, method = 'GET', data) => new Promise((ok, no) => uni.request({ url: BASE + url, method, data,
  header: { Authorization: 'Bearer ' + (uni.getStorageSync('token') || '') },
  success: r => { const b = r.data || {}; b.code === 0 ? ok(b.data) : no(b) }, fail: no }))
const dayKey = t => { const d = new Date(t); return d.getFullYear() + '-' + (d.getMonth() + 1) + '-' + d.getDate() }

// ---------- 登录与授权 ----------
export const hasConsent = () => !!uni.getStorageSync(K.consent)
export const getConsent = () => uni.getStorageSync(K.consent) || null
export function guard() { if (!hasConsent()) { uni.reLaunch({ url: '/pages/login/login' }); return false } return true }
export async function login(nickname = '王阿姨') {
  const consent = { agreedAt: Date.now(), version: CONSENT_VERSION }
  if (!USE_LOCAL) {
    // #ifdef MP-WEIXIN
    const { code } = await uni.login({ provider: 'weixin' })
    const r = await req('/api/auth/login', 'POST', { code, consent })
    set('token', r.token)
    // #endif
  }
  set(K.consent, consent)
  if (!get(K.profile, null)) set(K.profile, { nickname, since: Date.now() })
  if (!uni.getStorageSync(K.seeded)) seedDemo()
}
export const getProfile = () => get(K.profile, { nickname: '用户', since: Date.now() })

// ---------- 血压 ----------
export async function listRecords() { if (!USE_LOCAL) return req('/api/bp'); return get(K.rec).sort((a, b) => b.time - a.time) }
export async function getRecord(id) { return (await listRecords()).find(r => r.id === id) }
export async function addRecord(r) {
  if (!USE_LOCAL) return req('/api/bp', 'POST', r)
  const l = get(K.rec); l.push({ id: uid(), time: Date.now(), ...r }); set(K.rec, l); return {}
}
export async function updateRecord(id, r) {
  if (!USE_LOCAL) return req('/api/bp/' + id, 'PUT', r)
  set(K.rec, get(K.rec).map(x => x.id === id ? { ...x, ...r } : x)); return {}
}
export async function deleteRecord(id) {
  if (!USE_LOCAL) return req('/api/bp/' + id, 'DELETE')
  set(K.rec, get(K.rec).filter(x => x.id !== id)); return {}
}
// 校验只拦明显输错，措辞中性
export function validate({ sys, dia, hr }) {
  if (!sys || sys < 40 || sys > 300) return '收缩压请填写 40–300 之间的数字'
  if (!dia || dia < 20 || dia > 200) return '舒张压请填写 20–200 之间的数字'
  if (dia >= sys) return '舒张压应小于收缩压，请再核对一下'
  if (hr && (hr < 20 || hr > 250)) return '心率请填写 20–250 之间的数字，或留空'
  return ''
}

// ---------- 用药 ----------
export async function listMedDefs() { return USE_LOCAL ? get(K.meds) : req('/api/meds') }
export async function saveMed(m) {
  if (!USE_LOCAL) return m.id ? req('/api/meds/' + m.id, 'PUT', m) : req('/api/meds', 'POST', m)
  const l = get(K.meds); const times = [...m.times].sort()
  if (m.id) set(K.meds, l.map(x => x.id === m.id ? { ...x, ...m, times } : x)); else { l.push({ ...m, times, id: uid() }); set(K.meds, l) }
  return {}
}
export async function deleteMed(id) {
  if (!USE_LOCAL) return req('/api/meds/' + id, 'DELETE')
  set(K.meds, get(K.meds).filter(x => x.id !== id)); return {}
}
// 今日用药：每种药按时间点拆成一行 { id, name, dose, time, done }
export async function listMeds() {
  if (!USE_LOCAL) { const meds = await req('/api/meds'); return meds.flatMap(m => (m.slots || []).map(s => ({ ...m, time: s.time, done: s.done }))).sort((a, b) => a.time.localeCompare(b.time)) }
  const tk = dayKey(Date.now())
  const done = get(K.checkin).filter(c => dayKey(c.at) === tk).map(c => c.medId + '@' + c.time)
  return get(K.meds).flatMap(m => (m.times || []).map(t => ({ ...m, time: t, done: done.includes(m.id + '@' + t) }))).sort((a, b) => a.time.localeCompare(b.time))
}
export async function checkin(medId, time) {
  if (!USE_LOCAL) return req('/api/meds/' + medId + '/checkin', 'POST', { time })
  const l = get(K.checkin); l.push({ medId, time, at: Date.now() }); set(K.checkin, l); return {}
}
export async function listCheckins() {
  if (!USE_LOCAL) return req('/api/meds/checkins')
  const meds = get(K.meds); const since = Date.now() - 30 * 864e5
  return get(K.checkin).filter(c => c.at >= since).map(c => ({ ...c, name: (meds.find(m => m.id === c.medId) || {}).name || '已删除的药品' })).sort((a, b) => b.at - a.at)
}

// ---------- 测量提醒 ----------
const DEF_REMIND = { morning: { on: true, time: '08:00' }, evening: { on: true, time: '20:00' } }
export async function getRemind() { return USE_LOCAL ? get(K.remind, DEF_REMIND) : req('/api/reminders/measure') }
export async function saveRemind(r) {
  if (!USE_LOCAL) return req('/api/reminders/measure', 'PUT', r)
  set(K.remind, r)
  // #ifdef MP-WEIXIN
  const TMPL = [] // TODO 订阅消息模板 ID；真正推送需要用户授权订阅消息
  if (TMPL.length) uni.requestSubscribeMessage({ tmplIds: TMPL })
  // #endif
  return {}
}

// ---------- 家人共享（本地模拟） ----------
export async function createInvite() {
  if (!USE_LOCAL) return req('/api/family/invite', 'POST')
  const inv = { code: String(Math.floor(100000 + Math.random() * 900000)), expireAt: Date.now() + 864e5 }
  set(K.invite, inv); return inv
}
export const getInvite = () => { const v = get(K.invite, null); return v && v.expireAt > Date.now() ? v : null }
export async function listFamily() { return USE_LOCAL ? get(K.family) : (await req('/api/family')).members }
export async function unbindFamily(id) {
  if (!USE_LOCAL) return req('/api/family/' + id, 'DELETE')
  set(K.family, get(K.family).filter(x => x.id !== id)); return {}
}

// ---------- 账号 ----------
export async function exportData() {
  if (!USE_LOCAL) return req('/api/account/export')
  return { exportedAt: new Date().toISOString(), consent: getConsent(), profile: getProfile(), records: get(K.rec), meds: get(K.meds), checkins: get(K.checkin), reminders: get(K.remind, DEF_REMIND), family: get(K.family) }
}
export async function deleteAccount() {
  if (!USE_LOCAL) await req('/api/account', 'DELETE')
  uni.clearStorageSync()
}
export function clearDemo() { set(K.rec, []); set(K.meds, []); set(K.checkin, []); set(K.family, []); set(K.seeded, 1) }

// ---------- 示例数据 ----------
export function seedDemo() {
  const now = new Date(); const recs = []; const ck = []
  let s = 7
  const rnd = (a, b) => { s = (s * 9301 + 49297) % 233280; return a + Math.floor(s / 233280 * (b - a + 1)) }
  const tags = ['头晕', '乏力']
  for (let i = 10; i >= 0; i--) {
    for (const [h, m] of [[8, rnd(0, 25)], [20, rnd(10, 45)]]) {
      const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() - i, h, m)
      if (d > now) continue
      recs.push({ id: uid() + recs.length, time: d.getTime(), sys: rnd(118, 136), dia: rnd(74, 88), hr: rnd(64, 80), tags: rnd(0, 6) === 0 ? [tags[rnd(0, 1)]] : [], note: '' })
    }
  }
  const meds = [{ id: 'm1', name: '氨氯地平片', dose: '5mg · 1片', times: ['08:00'] }, { id: 'm2', name: '缬沙坦胶囊', dose: '80mg · 1粒', times: ['20:00'] }]
  for (let i = 10; i >= 0; i--) for (const m of meds) {
    const [h, mm] = m.times[0].split(':').map(Number)
    const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() - i, h, mm + rnd(0, 20))
    if (d <= now && rnd(0, 8) > 0) ck.push({ medId: m.id, time: m.times[0], at: d.getTime() })
  }
  set(K.rec, recs); set(K.meds, meds); set(K.checkin, ck)
  set(K.family, [{ id: 'f1', name: '小明', relation: '儿子', boundAt: Date.now() - 6 * 864e5 }, { id: 'f2', name: '小红', relation: '女儿', boundAt: Date.now() - 3 * 864e5 }])
  set(K.seeded, 1)
}

// ---------- 工具 ----------
const p2 = n => String(n).padStart(2, '0')
export const WEEK = ['日', '一', '二', '三', '四', '五', '六']
export const fmtTime = t => { const d = new Date(t); return `${d.getMonth() + 1}月${d.getDate()}日 ${p2(d.getHours())}:${p2(d.getMinutes())}` }
export const hhmm = t => { const d = new Date(t); return p2(d.getHours()) + ':' + p2(d.getMinutes()) }
export const isToday = t => dayKey(t) === dayKey(Date.now())
export function dayLabel(t) {
  const d = new Date(t), n = new Date(); const diff = Math.round((new Date(n.getFullYear(), n.getMonth(), n.getDate()) - new Date(d.getFullYear(), d.getMonth(), d.getDate())) / 864e5)
  return diff === 0 ? '今天' : diff === 1 ? '昨天' : `${d.getMonth() + 1}月${d.getDate()}日`
}
export function weekStart() { const n = new Date(); const w = (n.getDay() + 6) % 7; return new Date(n.getFullYear(), n.getMonth(), n.getDate() - w).getTime() }
export { dayKey }
