// 数据层：USE_LOCAL=true 时全部存本地（uni.setStorageSync）；false 时请求 Go 后端（见仓库 server/README.md）。
// 通过 Vite 环境变量切换：VITE_USE_LOCAL=false VITE_API_BASE=https://api.example.com（默认本地模式）
const ENV = (typeof import.meta !== 'undefined' && import.meta.env) || {}
export const USE_LOCAL = String(ENV.VITE_USE_LOCAL ?? 'true') !== 'false'
const BASE = (ENV.VITE_API_BASE || '').replace(/\/$/, '')
// 订阅消息模板 ID（公众平台申请后填入，与后端 WX_TPL_* 对应）
export const SUBSCRIBE_TMPL_IDS = []
const K = { rec: 'bp_records', consent: 'health_consent', meds: 'meds', checkin: 'med_checkin', remind: 'measure_remind', family: 'family_members', invite: 'family_invite', profile: 'profile', seeded: 'demo_seeded' }
export const CONSENT_VERSION = '1.0'
const get = (k, d = []) => { const v = uni.getStorageSync(k); return v === '' || v == null ? d : v }
const set = (k, v) => uni.setStorageSync(k, v)
const uid = () => Date.now().toString(36) + Math.random().toString(36).slice(2, 6)
let relaunching = false
const req = (url, method = 'GET', data) => new Promise((ok, no) => uni.request({ url: BASE + url, method, data,
  header: { Authorization: 'Bearer ' + (uni.getStorageSync('token') || ''), 'Content-Type': 'application/json' },
  success: r => {
    const b = r.data || {}
    if (r.statusCode === 401 || b.code === 401) {
      uni.removeStorageSync('token'); uni.removeStorageSync(K.consent)
      if (!relaunching) { relaunching = true; uni.reLaunch({ url: '/pages/login/login', complete: () => { relaunching = false } }) }
      return no(b)
    }
    if (b.code === 0) { if (b.data && b.data.needSubscribe) askSubscribe(b.data.needSubscribe); ok(b.data) }
    else { uni.showToast({ title: b.message || '请求失败', icon: 'none' }); no(b) }
  },
  fail: e => { uni.showToast({ title: '网络异常，请稍后再试', icon: 'none' }); no(e) } }))
// 后端提示剩余推送次数不足时：先弹窗说明，再请求订阅消息授权，接受的上报 /api/subscribe
let asking = false
function askSubscribe(ns) {
  const ids = [...new Set([ns.templateId, ...SUBSCRIBE_TMPL_IDS].filter(Boolean))]
  if (asking || !ids.length) return
  asking = true
  uni.showModal({ title: '开启提醒', content: '允许接收微信提醒后，到点会通知你测量或服药。', confirmText: '允许', cancelText: '暂不', confirmColor: '#10C495',
    success: m => {
      if (!m.confirm) return
      uni.requestSubscribeMessage({ tmplIds: ids, success: r => {
        const accepted = r[ns.templateId] === 'accept' ? [ns.kind] : []
        if (accepted.length) req('/api/subscribe', 'POST', { accepted }).catch(() => {})
      } })
    }, complete: () => { asking = false } })
}
const ms = t => typeof t === 'number' ? t : Date.parse(t)
const toRec = r => ({ ...r, time: ms(r.time), tags: r.tags || [], note: r.note || '' })
const recBody = r => { const b = { sys: r.sys, dia: r.dia, hr: r.hr || 0, tags: r.tags || [], note: r.note || '' }; if (r.time) b.time = new Date(r.time).toISOString(); return b }
const dayKey = t => { const d = new Date(t); return d.getFullYear() + '-' + (d.getMonth() + 1) + '-' + d.getDate() }

// ---------- 登录与授权 ----------
export const hasConsent = () => !!uni.getStorageSync(K.consent)
export const getConsent = () => uni.getStorageSync(K.consent) || null
export function guard() { if (!hasConsent()) { uni.reLaunch({ url: '/pages/login/login' }); return false } return true }
export async function login(nickname = '王阿姨') {
  const consent = { agreedAt: Date.now(), version: CONSENT_VERSION }
  if (!USE_LOCAL) {
    let code = 'h5-dev' // H5 无微信登录，仅供后端 APP_ENV=dev 联调
    // #ifdef MP-WEIXIN
    code = (await uni.login({ provider: 'weixin' })).code
    // #endif
    const r = await req('/api/auth/login', 'POST', { code, nickname, consent })
    set('token', r.token)
    set(K.consent, consent)
    if (!get(K.profile, null)) set(K.profile, { nickname, since: Date.now() })
    return
  }
  set(K.consent, consent)
  if (!get(K.profile, null)) set(K.profile, { nickname, since: Date.now() })
  if (!uni.getStorageSync(K.seeded)) seedDemo()
}
export const getProfile = () => get(K.profile, { nickname: '用户', since: Date.now() })

// ---------- 血压 ----------
export async function listRecords() { if (!USE_LOCAL) return (await req('/api/bp')).map(toRec); return get(K.rec).sort((a, b) => b.time - a.time) }
export async function getRecord(id) { return (await listRecords()).find(r => r.id === id) }
export async function addRecord(r) {
  if (!USE_LOCAL) { const d = await req('/api/bp', 'POST', recBody(r)); return { ...d, record: d.record && toRec(d.record) } }
  const l = get(K.rec); l.push({ id: uid(), time: Date.now(), ...r }); set(K.rec, l); return {}
}
export async function updateRecord(id, r) {
  if (!USE_LOCAL) return req('/api/bp/' + id, 'PUT', recBody(r))
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
  if (!USE_LOCAL) { const b = { name: m.name, dose: m.dose || '', times: [...m.times].sort() }; return m.id ? req('/api/meds/' + m.id, 'PUT', b) : req('/api/meds', 'POST', b) }
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
  if (!USE_LOCAL) {
    const [cks, meds] = await Promise.all([req('/api/meds/checkins'), req('/api/meds')])
    return cks.map(c => ({ medId: c.medId, time: c.slot, at: ms(c.at), name: (meds.find(m => m.id === c.medId) || {}).name || '已删除的药品' })).sort((a, b) => b.at - a.at)
  }
  const meds = get(K.meds); const since = Date.now() - 30 * 864e5
  return get(K.checkin).filter(c => c.at >= since).map(c => ({ ...c, name: (meds.find(m => m.id === c.medId) || {}).name || '已删除的药品' })).sort((a, b) => b.at - a.at)
}

// ---------- 测量提醒 ----------
const DEF_REMIND = { morning: { on: true, time: '08:00' }, evening: { on: true, time: '20:00' } }
// 后端只有一个 enabled 开关：本地结构 { morning:{on,time}, evening:{on,time} } 与之互转（关闭的时段时间传空串）
export async function getRemind() {
  if (USE_LOCAL) return get(K.remind, DEF_REMIND)
  const m = await req('/api/reminders/measure')
  if (!m || (!m.morning && !m.evening)) return DEF_REMIND
  return { morning: { on: !!(m.enabled && m.morning), time: m.morning || '08:00' }, evening: { on: !!(m.enabled && m.evening), time: m.evening || '20:00' } }
}
export async function saveRemind(r) {
  if (!USE_LOCAL) {
    await req('/api/reminders/measure', 'PUT', { enabled: r.morning.on || r.evening.on, morning: r.morning.on ? r.morning.time : '', evening: r.evening.on ? r.evening.time : '' })
    return {}
  }
  set(K.remind, r)
  // #ifdef MP-WEIXIN
  if (SUBSCRIBE_TMPL_IDS.length) uni.requestSubscribeMessage({ tmplIds: SUBSCRIBE_TMPL_IDS })
  // #endif
  return {}
}

// ---------- 家人共享（本地模拟） ----------
export async function createInvite() {
  if (!USE_LOCAL) { const d = await req('/api/family/invite', 'POST'); const inv = { code: d.code, expireAt: ms(d.expiresAt) }; set(K.invite, inv); return inv }
  const inv = { code: String(Math.floor(100000 + Math.random() * 900000)), expireAt: Date.now() + 864e5 }
  set(K.invite, inv); return inv
}
export const getInvite = () => { const v = get(K.invite, null); return v && v.expireAt > Date.now() ? v : null }
// members：能看我的人（id 为绑定关系 id，仅用于解绑）
export async function listFamily() {
  if (USE_LOCAL) return get(K.family)
  return ((await req('/api/family')).members || []).map(f => ({ id: f.id, memberId: f.memberId, name: f.nickname || '家人', relation: '', boundAt: ms(f.createdAt) }))
}
// patients：我能看的人；查看用 patientId
export async function listPatients() { return USE_LOCAL ? [] : ((await req('/api/family')).patients || []).map(p => ({ id: p.id, patientId: p.patientId, name: p.nickname || '家人', boundAt: ms(p.createdAt) })) }
export async function listPatientRecords(pid) { return (await req('/api/family/patients/' + pid + '/bp')).map(toRec) }
export async function patientStats(pid, range = 'week') { return req('/api/family/patients/' + pid + '/bp/stats?range=' + range) }
export async function bindFamily(code) { return req('/api/family/bind', 'POST', { code }) }
export async function listTodos() { return USE_LOCAL ? [] : req('/api/todos') }
export async function doneTodo(id) { return USE_LOCAL ? {} : req('/api/todos/' + id + '/done', 'POST') }
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

// ---------- 就诊报告卡 ----------
// 与后端 GET /api/report 同结构：{ from, to, days, generatedAt, average, morning, evening, count, series[{date,morning,evening}], meds[], adherence, symptoms[], footer }
export const REPORT_FOOTER = '数据由用户自行记录，仅供就诊参考'
const ymd = d => d.getFullYear() + '-' + p2(d.getMonth() + 1) + '-' + p2(d.getDate())
export async function getReport(opt = {}) {
  if (!USE_LOCAL) {
    const q = opt.from && opt.to ? `from=${opt.from}&to=${opt.to}` : 'days=' + (opt.days || 30)
    const r = await req('/api/report?' + q)
    return { ...r, generatedAt: ms(r.generatedAt), footer: r.footer || REPORT_FOOTER }
  }
  const n = new Date(); let start, end = new Date(n.getFullYear(), n.getMonth(), n.getDate())
  if (opt.from && opt.to) { const [a, b] = [opt.from, opt.to].map(s => { const [y, m, d] = s.split('-').map(Number); return new Date(y, m - 1, d) }); start = a; end = b }
  else start = new Date(end.getFullYear(), end.getMonth(), end.getDate() - ((opt.days || 30) - 1))
  const ndays = Math.round((end - start) / 864e5) + 1
  if (ndays < 1 || ndays > 90) throw new Error('时间范围最长 90 天')
  const stop = new Date(end.getFullYear(), end.getMonth(), end.getDate() + 1).getTime()
  const recs = get(K.rec).filter(r => r.time >= start.getTime() && r.time < stop).sort((a, b) => a.time - b.time)
  const acc = () => ({ s: 0, d: 0, h: 0, hn: 0, n: 0 })
  const add = (x, b) => { x.s += b.sys; x.d += b.dia; x.n++; if (b.hr > 0) { x.h += b.hr; x.hn++ } }
  const r1 = v => Math.round(v * 10) / 10
  const fin = x => !x || !x.n ? null : { sys: r1(x.s / x.n), dia: r1(x.d / x.n), hr: x.hn ? r1(x.h / x.hn) : null, count: x.n }
  const all = acc(), am = acc(), pm = acc(), dAM = {}, dPM = {}, sym = {}
  for (const b of recs) {
    const t = new Date(b.time), k = ymd(t); add(all, b)
    const tg = t.getHours() < 12 ? (add(am, b), dAM) : (add(pm, b), dPM)
    add(tg[k] = tg[k] || acc(), b)
    for (const s of b.tags || []) sym[s] = (sym[s] || 0) + 1
  }
  const series = []
  for (let i = 0; i < ndays; i++) { const k = ymd(new Date(start.getFullYear(), start.getMonth(), start.getDate() + i)); series.push({ date: k, morning: fin(dAM[k]), evening: fin(dPM[k]) }) }
  const f = ymd(start), t = ymd(end); let planned = 0, taken = 0
  const ck = get(K.checkin).filter(c => { const k = ymd(new Date(c.at)); return k >= f && k <= t })
  const meds = get(K.meds).map(m => {
    const p = (m.times || []).length * ndays, tk = new Set(ck.filter(c => c.medId === m.id).map(c => ymd(new Date(c.at)) + '@' + c.time)).size
    planned += p; taken += tk
    return { name: m.name, dose: m.dose || '', times: m.times || [], planned: p, taken: tk, rate: p ? r1(tk * 100 / p) : null }
  })
  return { from: f, to: t, days: ndays, generatedAt: Date.now(), average: fin(all), morning: fin(am), evening: fin(pm), count: all.n, series, meds,
    adherence: { planned, taken, rate: planned ? r1(taken * 100 / planned) : null },
    symptoms: Object.entries(sym).map(([name, count]) => ({ name, count })).sort((a, b) => b.count - a.count), footer: REPORT_FOOTER }
}
// 报告卡上的姓名只存本地，不发给后端
export const getReportName = () => uni.getStorageSync('report_name') || ''
export const setReportName = v => uni.setStorageSync('report_name', v)
