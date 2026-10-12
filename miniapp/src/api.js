// 数据层：USE_LOCAL=true 时全部存本地（uni.setStorageSync）；false 时请求 Go 后端（见仓库 server/README.md）。
// 通过 Vite 环境变量切换：VITE_USE_LOCAL=false VITE_API_BASE=https://api.example.com（默认本地模式）
const ENV = (typeof import.meta !== 'undefined' && import.meta.env) || {}
export const USE_LOCAL = String(ENV.VITE_USE_LOCAL ?? 'true') !== 'false'
const BASE = (ENV.VITE_API_BASE || '').replace(/\/$/, '')
// 订阅消息模板 ID（公众平台申请后填入，与后端 WX_TPL_* 对应）
export const SUBSCRIBE_TMPL_IDS = []
const K = { rec: 'bp_records', consent: 'health_consent', meds: 'meds', checkin: 'med_checkin', remind: 'measure_remind', family: 'family_members', invite: 'family_invite', profile: 'profile', seeded: 'demo_seeded', visits: 'visits', role: 'role', fpatients: 'family_patients', skip: 'med_skip' }
export const CONSENT_VERSION = '1.0'
const get = (k, d = []) => { const v = uni.getStorageSync(k); return v === '' || v == null ? d : v }
const set = (k, v) => uni.setStorageSync(k, v)
const uid = () => Date.now().toString(36) + Math.random().toString(36).slice(2, 6)
let relaunching = false
const req = (url, method = 'GET', data, quiet) => new Promise((ok, no) => uni.request({ url: BASE + url, method, data,
  header: { Authorization: 'Bearer ' + (uni.getStorageSync('token') || ''), 'Content-Type': 'application/json' },
  success: r => {
    const b = r.data || {}
    if (r.statusCode === 401 || b.code === 401) {
      uni.removeStorageSync('token'); uni.removeStorageSync(K.consent)
      if (!relaunching) { relaunching = true; uni.reLaunch({ url: '/pages/login/login', complete: () => { relaunching = false } }) }
      return no(b)
    }
    if (b.code === 0) { if (b.data && b.data.needSubscribe) askSubscribe(b.data.needSubscribe); ok(b.data) }
    else { if (!(quiet && quiet.includes(b.code))) uni.showToast({ title: b.message || '请求失败', icon: 'none' }); no(b) }
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
// 本地打卡记录：{ medId, time(计划时段 HH:MM), date(计划日期 YYYY-MM-DD), at(打卡/补记操作时间), makeup?, recordedAt?, fromStatus? }
// 本地「没吃」标记（V1.3）：med_skip [{ medId, date, time, at }]
// 药品软删除（deletedAt），停用/删除后其 7 天内历史时段仍可补记（PRD 5.8）
export const MAKEUP_DAYS = 7 // 可补记天数（含今天）【可调整】
export const MISS_WINDOW_MIN = 120 // 漏服判定窗口：计划时间后 2 小时仍未打卡视为漏服【可调整】
const live = l => l.filter(m => !m.deletedAt)
const slotDate = c => c.date || ymd(new Date(c.at))
export async function listMedDefs() { return USE_LOCAL ? live(get(K.meds)) : req('/api/meds') }
export async function saveMed(m) {
  if (!USE_LOCAL) { const b = { name: m.name, dose: m.dose || '', times: [...m.times].sort() }; return m.id ? req('/api/meds/' + m.id, 'PUT', b) : req('/api/meds', 'POST', b) }
  const l = get(K.meds); const times = [...m.times].sort()
  if (m.id) set(K.meds, l.map(x => x.id === m.id ? { ...x, ...m, times } : x)); else { l.push({ ...m, times, id: uid(), createdAt: Date.now() }); set(K.meds, l) }
  return {}
}
export async function deleteMed(id) {
  if (!USE_LOCAL) return req('/api/meds/' + id, 'DELETE')
  set(K.meds, get(K.meds).map(x => x.id === id ? { ...x, deletedAt: Date.now() } : x)); return {}
}
// 由用药计划 + 打卡记录推算时段状态：taken 已服 / skipped 未服（主动点了没吃） / missed 漏服 / pending 待服（窗口内） / future 未到时间
export function computeSlots(meds, cks, days = MAKEUP_DAYS, now = Date.now(), skips = []) {
  const n = new Date(now), out = [], map = {}, sk = {}
  for (const k of skips) sk[k.medId + '@' + k.date + '@' + k.time] = 1
  for (const c of cks) map[c.medId + '@' + slotDate(c) + '@' + c.time] = c
  const lim = new Date(n.getFullYear(), n.getMonth(), n.getDate() - (MAKEUP_DAYS - 1)).getTime()
  for (let i = 0; i < days; i++) {
    const d0 = new Date(n.getFullYear(), n.getMonth(), n.getDate() - i), date = ymd(d0)
    for (const m of meds) for (const t of (m.times || [])) {
      const [h, mi] = t.split(':').map(Number), sched = new Date(d0.getFullYear(), d0.getMonth(), d0.getDate(), h, mi).getTime()
      if (m.createdAt && sched < new Date(new Date(m.createdAt).setHours(0, 0, 0, 0)).getTime()) continue
      if (m.deletedAt && sched > m.deletedAt) continue
      const c = map[m.id + '@' + date + '@' + t]
      const raw = sched > now ? 'future' : now - sched < MISS_WINDOW_MIN * 6e4 ? 'pending' : 'missed'
      const status = c ? 'taken' : sk[m.id + '@' + date + '@' + t] && raw !== 'future' ? 'skipped' : raw, inWin = sched >= lim
      out.push({ medId: m.id, name: m.name, dose: m.dose || '', date, time: t, sched, status, checkedAt: c ? c.at : null, makeup: !!(c && c.makeup), recordedAt: c && c.makeup ? (c.recordedAt || c.at) : null, deleted: !!m.deletedAt,
        canMakeup: (status === 'missed' || status === 'skipped') && inWin, canUndo: !!(c && c.makeup) && inWin,
        canSkip: (status === 'pending' || status === 'missed') && inWin, canUnskip: status === 'skipped' && inWin })
    }
  }
  return out.sort((a, b) => b.sched - a.sched)
}
const fromSlot = s => ({ ...s, status: s.status === 'upcoming' ? 'future' : s.status, sched: ms(s.scheduledAt), checkedAt: s.checkedAt ? ms(s.checkedAt) : null, recordedAt: s.recordedAt ? ms(s.recordedAt) : null, makeup: !!s.isMakeup, canMakeup: !!s.canMakeup, canUndo: !!s.canRevoke, canSkip: !!s.canSkip, canUnskip: !!s.canUnskip })
const RO = s => ({ ...s, canMakeup: false, canUndo: false, canSkip: false, canUnskip: false })
// 时段列表（含状态、补记标签），默认最近 7 天
export async function listSlots(days = MAKEUP_DAYS) {
  if (!USE_LOCAL) return ((await req('/api/meds/history?days=' + days)) || []).map(fromSlot)
  return computeSlots(get(K.meds), get(K.checkin), days, Date.now(), get(K.skip))
}
export async function makeupCount() { if (!USE_LOCAL) return ((await req('/api/meds/makeup')) || {}).count || 0; return (await listMakeupSlots()).length }
export async function listMakeupSlots() { return (await listSlots(MAKEUP_DAYS)).filter(s => s.canMakeup) }
// 补记：只能补记最近 7 天内漏服或未服的计划时段（不记录实际服药时间）
export async function makeupCheckin(s) {
  if (!USE_LOCAL) return req('/api/meds/' + s.medId + '/makeup', 'POST', { date: s.date, time: s.time })
  const cur = computeSlots(get(K.meds), get(K.checkin), MAKEUP_DAYS, Date.now(), get(K.skip)).find(x => x.medId === s.medId && x.date === s.date && x.time === s.time)
  if (!cur || !cur.canMakeup) throw new Error('这个时段不能补记')
  const now = Date.now(), l = get(K.checkin)
  l.push({ medId: s.medId, time: s.time, date: s.date, at: now, makeup: true, recordedAt: now, fromStatus: cur.status }); set(K.checkin, l)
  const log = get('makeup_log'); log.push({ op: 'makeup', medId: s.medId, date: s.date, time: s.time, at: now }); set('makeup_log', log)
  return {}
}
export async function undoMakeup(s) {
  if (!USE_LOCAL) return req('/api/meds/' + s.medId + '/makeup?date=' + s.date + '&time=' + encodeURIComponent(s.time), 'DELETE')
  set(K.checkin, get(K.checkin).filter(c => !(c.makeup && c.medId === s.medId && slotDate(c) === s.date && c.time === s.time)))
  const log = get('makeup_log'); log.push({ op: 'revoke', medId: s.medId, date: s.date, time: s.time, at: Date.now() }); set('makeup_log', log)
  return {}
}
// 没吃（未服）：只能标已到点、未打卡的时段（待服或漏服），7 天内；可撤销。不触发家人漏服通知
export async function skipSlot(s) {
  if (!USE_LOCAL) return req('/api/meds/' + s.medId + '/skip', 'POST', { date: s.date, time: s.time })
  const cur = computeSlots(get(K.meds), get(K.checkin), MAKEUP_DAYS, Date.now(), get(K.skip)).find(x => x.medId === s.medId && x.date === s.date && x.time === s.time)
  if (!cur || !cur.canSkip) throw new Error('这个时段不能标记')
  const l = get(K.skip); l.push({ medId: s.medId, date: s.date, time: s.time, at: Date.now() }); set(K.skip, l); return {}
}
export async function unskipSlot(s) {
  if (!USE_LOCAL) return req('/api/meds/' + s.medId + '/skip?date=' + s.date + '&time=' + encodeURIComponent(s.time), 'DELETE')
  set(K.skip, get(K.skip).filter(k => !(k.medId === s.medId && k.date === s.date && k.time === s.time))); return {}
}
// 今日用药：每种药按时间点拆成一行 { id, name, dose, time, done, status, makeup }
export async function listMeds() {
  if (!USE_LOCAL) { const td = ymd(new Date()); return (await listSlots(1)).filter(s => s.date === td && !s.deleted).map(s => ({ ...s, id: s.medId, done: s.status === 'taken' })).sort((a, b) => a.time.localeCompare(b.time)) }
  const defs = get(K.meds), td = ymd(new Date())
  return computeSlots(defs, get(K.checkin), 1, Date.now(), get(K.skip)).filter(s => s.date === td && !s.deleted).map(s => ({ ...defs.find(m => m.id === s.medId), ...s, id: s.medId, done: s.status === 'taken' })).sort((a, b) => a.time.localeCompare(b.time))
}
export async function checkin(medId, time) {
  if (!USE_LOCAL) return req('/api/meds/' + medId + '/checkin', 'POST', { time })
  const l = get(K.checkin); l.push({ medId, time, date: ymd(new Date()), at: Date.now() }); set(K.checkin, l); return {}
}
export async function listCheckins() {
  if (!USE_LOCAL) {
    const [cks, meds] = await Promise.all([req('/api/meds/checkins'), req('/api/meds')])
    return cks.map(c => ({ medId: c.medId, time: c.slot, at: ms(c.at), makeup: !!c.isMakeup, name: (meds.find(m => m.id === c.medId) || {}).name || '已删除的药品' })).sort((a, b) => b.at - a.at)
  }
  const meds = get(K.meds); const since = Date.now() - 30 * 864e5
  return get(K.checkin).filter(c => c.at >= since).map(c => ({ ...c, name: (meds.find(m => m.id === c.medId) || {}).name || '已删除的药品' })).sort((a, b) => b.at - a.at)
}

// ---------- 复诊日提醒（PRD 5.9） ----------
// 本地：visits [{ id, date, hospital, department, note, remindTime, status: upcoming|past|cancelled, createdAt, updatedAt }]
export const VISIT_TMPL_ID = '' // 订阅消息模板「复诊提醒」visit_remind，审核通过后填入
export const VISIT_REMIND_DEFAULT = '19:00' // 复诊前一天提醒时间【可调整】
const parseYmd = s => { const [y, m, d] = s.split('-').map(Number); return new Date(y, m - 1, d) }
export const visitRemindAt = v => { const d = parseYmd(v.date); const [h, m] = (v.remindTime || VISIT_REMIND_DEFAULT).split(':').map(Number); return new Date(d.getFullYear(), d.getMonth(), d.getDate() - 1, h, m).getTime() }
export const daysUntil = date => Math.round((parseYmd(date) - new Date(new Date().setHours(0, 0, 0, 0))) / 864e5)
function rollVisits() { // 复诊日过后自动改为「已过」
  const td = ymd(new Date()); let changed = false
  const l = get(K.visits).map(v => { if (v.status === 'upcoming' && v.date < td) { changed = true; return { ...v, status: 'past', updatedAt: Date.now() } } return v })
  if (changed) { set(K.visits, l); set('visit_ask_next', 1) }
  return l
}
const toVisit = v => v && { ...v, date: v.date || v.visitDate, remindTime: v.remindTime || VISIT_REMIND_DEFAULT }
const ymdOf = s => typeof s === 'string' ? s.slice(0, 10) : s
export async function getVisit() {
  if (!USE_LOCAL) return toVisit(((await req('/api/visits')) || {}).upcoming || null)
  return rollVisits().find(v => v.status === 'upcoming') || null
}
export async function listVisitHistory() {
  if (!USE_LOCAL) return (((await req('/api/visits')) || {}).history || []).map(toVisit)
  return rollVisits().filter(v => v.status !== 'upcoming').sort((a, b) => b.date.localeCompare(a.date) || b.updatedAt - a.updatedAt)
}
// 保存复诊：v.id 存在则修改；否则新增。已有即将到来的复诊且未传 replace 时抛出 { conflict: 原复诊 }；replace=true 时原记录改为「已取消」进入历史
export async function saveVisit(v, replace = false) {
  const b = { date: v.date, hospital: (v.hospital || '').slice(0, 30), department: (v.department || '').slice(0, 20), note: (v.note || '').slice(0, 100), remindTime: v.remindTime || VISIT_REMIND_DEFAULT }
  // #ifdef MP-WEIXIN
  if (VISIT_TMPL_ID) await new Promise(ok => uni.requestSubscribeMessage({ tmplIds: [VISIT_TMPL_ID], success: r => { if (!USE_LOCAL && r[VISIT_TMPL_ID] === 'accept') req('/api/subscribe', 'POST', { accepted: ['visit_remind'] }).catch(() => {}) }, complete: ok }))
  // #endif
  if (!USE_LOCAL) {
    const rb = { visitDate: b.date, hospital: b.hospital, department: b.department, note: b.note, remindTime: b.remindTime }
    if (v.id) return req('/api/visits/' + v.id, 'PUT', rb)
    try { return await req('/api/visits', 'POST', { ...rb, replace }, [409]) } catch (e) { if (e && e.code === 409) throw { conflict: toVisit(e.data) || {} }; throw e }
  }
  const now = Date.now(); let l = rollVisits()
  const td = ymd(new Date()), mx = ymd(new Date(Date.now() + 365 * 864e5))
  if (b.date < td || b.date > mx) throw new Error('请选择今天到一年内的日期')
  const cur = l.find(x => x.status === 'upcoming')
  if (!v.id && cur && !replace) throw { conflict: cur }
  if (v.id) l = l.map(x => x.id === v.id ? { ...x, ...b, updatedAt: now } : x)
  else { l = l.map(x => x.status === 'upcoming' ? { ...x, status: 'cancelled', updatedAt: now } : x); l.push({ ...b, id: uid(), status: 'upcoming', createdAt: now, updatedAt: now }) }
  set(K.visits, l); uni.removeStorageSync('visit_ask_next'); return {}
}
export async function cancelVisit(id) {
  if (!USE_LOCAL) return req('/api/visits/' + id + '/cancel', 'POST')
  set(K.visits, get(K.visits).map(x => x.id === id ? { ...x, status: 'cancelled', updatedAt: Date.now() } : x)); return {}
}
export async function deleteVisit(id) {
  if (!USE_LOCAL) return req('/api/visits/' + id, 'DELETE')
  set(K.visits, get(K.visits).filter(x => x.id !== id)); return {}
}
// 复诊日过后，下次打开时轻提示一次
export const shouldAskNextVisit = () => !!uni.getStorageSync('visit_ask_next')
export const dismissAskNextVisit = () => uni.removeStorageSync('visit_ask_next')

// ---------- 测量提醒 ----------
const DEF_REMIND = { morning: { on: true, time: '08:00' }, evening: { on: true, time: '20:00' } }
// 后端：enabled 总开关 + morningEnabled / eveningEnabled 分开关（未返回时视为开）
export async function getRemind() {
  if (USE_LOCAL) return get(K.remind, DEF_REMIND)
  const m = await req('/api/reminders/measure')
  if (!m || (!m.morning && !m.evening)) return DEF_REMIND
  return { morning: { on: !!(m.enabled && m.morning && m.morningEnabled !== false), time: m.morning || '08:00' }, evening: { on: !!(m.enabled && m.evening && m.eveningEnabled !== false), time: m.evening || '20:00' } }
}
export async function saveRemind(r) {
  if (!USE_LOCAL) {
    await req('/api/reminders/measure', 'PUT', { enabled: r.morning.on || r.evening.on, morningEnabled: r.morning.on, eveningEnabled: r.evening.on, morning: r.morning.time, evening: r.evening.time })
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
export async function listPatients() {
  if (USE_LOCAL) return get(K.fpatients).map(p => ({ id: p.id, patientId: p.id, name: p.name, relation: p.relation, boundAt: p.boundAt }))
  return ((await req('/api/family')).patients || []).map(p => ({ id: p.id, patientId: p.patientId, name: p.nickname || '家人', boundAt: ms(p.createdAt) }))
}
const noNote = r => ({ ...r, note: '' }) // 家人端不展示备注文字
export async function listPatientRecords(pid) {
  if (USE_LOCAL) { const p = get(K.fpatients).find(x => x.id === pid); return p ? p.records.map(noNote).sort((a, b) => b.time - a.time) : [] }
  return (await req('/api/family/patients/' + pid + '/bp')).map(toRec).map(noNote)
}
export async function patientStats(pid, range = 'week') { return req('/api/family/patients/' + pid + '/bp/stats?range=' + range) }
// 家人只读：服药历史（含补记标签，无补记/撤销能力）
export async function listPatientSlots(pid, days = MAKEUP_DAYS) {
  if (USE_LOCAL) { const p = get(K.fpatients).find(x => x.id === pid); return p ? computeSlots(p.meds, p.checkins, days, Date.now(), p.skips || []).map(RO) : [] }
  return ((await req('/api/family/patients/' + pid + '/meds/history?days=' + days)) || []).map(fromSlot).map(RO)
}
// 家人只读：仅即将到来的复诊日期、医院、科室（不含备注与历史）
export async function getPatientVisit(pid) {
  let v
  if (USE_LOCAL) { const p = get(K.fpatients).find(x => x.id === pid); v = p && p.visit && p.visit.date >= ymd(new Date()) ? p.visit : null }
  else v = toVisit(await req('/api/family/patients/' + pid + '/visit'))
  return v && v.date ? { date: ymdOf(v.date), hospital: v.hospital || '', department: v.department || '' } : null
}
const listPatientsSync = () => get(K.fpatients).map(p => ({ name: p.name, boundAt: p.boundAt }))
// 家人站内消息：漏服/漏测通知，补记后追加「已于……补记」（不撤回、不另发）
export async function listFamilyMessages(pid) {
  if (!USE_LOCAL) return ((await req('/api/family/messages')) || []).filter(m => !pid || m.patientId === pid).map(m => ({ ...m, createdAt: ms(m.createdAt), makeupNotes: m.makeupNotes || [] }))
  const p = get(K.fpatients).find(x => x.id === pid); if (!p) return []
  // 本地模拟：近 7 天漏服过的时段各一条（补记的追加说明）；标了「没吃」的时段不通知
  const out = []
  for (const sl of computeSlots(p.meds, p.checkins.filter(c => !c.makeup), MAKEUP_DAYS)) if (sl.status === 'missed' && !(p.skips || []).some(k => k.medId === sl.medId && k.date === sl.date && k.time === sl.time)) {
    const mk = p.checkins.find(c => c.makeup && c.medId === sl.medId && slotDate(c) === sl.date && c.time === sl.time)
    const [, m, d] = sl.date.split('-').map(Number)
    out.push({ id: sl.medId + sl.date + sl.time, patientId: pid, date: sl.date, createdAt: sl.sched + MISS_WINDOW_MIN * 6e4, text: `${m}月${d}日 ${sl.time} 的服药尚未打卡`, makeupNotes: mk ? [`${sl.time} 的用药已于 ${fmtTime(mk.recordedAt || mk.at)} 补记`] : [] })
  }
  return out
}
export async function bindFamily(code) {
  if (!USE_LOCAL) return req('/api/family/bind', 'POST', { code })
  if (!/^\d{6}$/.test(code)) throw new Error('请输入 6 位邀请码')
  const l = get(K.fpatients); const n = l.length
  l.push(genPatient('p' + uid(), ['外婆', '外公', '姑姑', '舅舅'][n % 4], '', 31 + n * 7, 12)); set(K.fpatients, l); return {}
}
export async function unfollowPatient(id) {
  if (!USE_LOCAL) return req('/api/family/' + id, 'DELETE')
  set(K.fpatients, get(K.fpatients).filter(x => x.id !== id)); return {}
}
// ---------- 身份入口：patient 我是患者 / family 我是家人 ----------
export const getRole = () => uni.getStorageSync(K.role) || ''
export const setRole = r => uni.setStorageSync(K.role, r)
export const homeOf = r => r === 'family' ? '/pages/family/home' : '/pages/index/index'
export async function listTodos() { return USE_LOCAL ? [] : req('/api/todos') }
export async function doneTodo(id) { return USE_LOCAL ? {} : req('/api/todos/' + id + '/done', 'POST') }
export async function unbindFamily(id) {
  if (!USE_LOCAL) return req('/api/family/' + id, 'DELETE')
  set(K.family, get(K.family).filter(x => x.id !== id)); return {}
}

// ---------- 账号 ----------
export async function exportData() {
  if (!USE_LOCAL) return req('/api/account/export')
  return { exportedAt: new Date().toISOString(), consent: getConsent(), profile: getProfile(), records: get(K.rec), meds: get(K.meds), checkins: get(K.checkin), reminders: get(K.remind, DEF_REMIND), family: get(K.family), visits: get(K.visits), followedPatients: listPatientsSync() }
}
export async function deleteAccount() {
  if (!USE_LOCAL) await req('/api/account', 'DELETE')
  uni.clearStorageSync()
}
export function clearDemo() { set(K.rec, []); set(K.meds, []); set(K.checkin, []); set(K.skip, []); set(K.family, []); set(K.visits, []); set(K.fpatients, []); set(K.seeded, 1) }

// ---------- 示例数据 ----------
// 生成一位示例患者的数据：近 14 天血压、2 种药、打卡（含 1 次漏服、1 次补记、1 次未服）
function genData(seed) {
  const now = new Date(); const recs = []; const ck = []; const sk = []; let s = seed
  const rnd = (a, b) => { s = (s * 9301 + 49297) % 233280; return a + Math.floor(s / 233280 * (b - a + 1)) }
  const tags = ['头晕', '乏力']
  for (let i = 13; i >= 0; i--) for (const [h, m] of [[8, rnd(0, 25)], [20, rnd(10, 45)]]) {
    const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() - i, h, m)
    if (d > now) continue
    recs.push({ id: uid() + recs.length, time: d.getTime(), sys: rnd(118, 136), dia: rnd(74, 88), hr: rnd(64, 80), tags: rnd(0, 6) === 0 ? [tags[rnd(0, 1)]] : [], note: '' })
  }
  const created = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 14).getTime()
  const meds = [{ id: 'm1', name: '氨氯地平片', dose: '5mg · 1片', times: ['08:00'], createdAt: created }, { id: 'm2', name: '缬沙坦胶囊', dose: '80mg · 1粒', times: ['20:00'], createdAt: created }]
  for (let i = 13; i >= 0; i--) for (const m of meds) {
    const [h, mm] = m.times[0].split(':').map(Number)
    const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() - i, h, mm + rnd(0, 20)), date = ymd(d)
    if (d > now) continue
    if (i === 1 && m.id === 'm1') continue // 昨天早上：漏服，可补记
    if (i === 3 && m.id === 'm1') { sk.push({ medId: m.id, date, time: m.times[0], at: d.getTime() + 36e5 }); continue } // 3 天前早上：未服（没吃）
    if (i === 2 && m.id === 'm2') { const at = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1, 21, 15).getTime(); ck.push({ medId: m.id, time: m.times[0], date, at, makeup: true, recordedAt: at, fromStatus: 'missed' }); continue }
    if (i > 7 && rnd(0, 8) === 0) continue
    ck.push({ medId: m.id, time: m.times[0], date, at: d.getTime() })
  }
  return { recs, meds, ck, sk }
}
const dAfter = n => { const d = new Date(); return ymd(new Date(d.getFullYear(), d.getMonth(), d.getDate() + n)) }
function genPatient(id, name, relation, seed, visitIn) {
  const { recs, meds, ck, sk } = genData(seed)
  return { id, name, relation, boundAt: Date.now() - 20 * 864e5, records: recs, meds, checkins: ck, skips: sk, visit: visitIn != null ? { date: dAfter(visitIn), hospital: '市第一人民医院', department: '心内科', note: '带上药盒' } : null }
}
export function seedDemo() {
  const { recs, meds, ck, sk } = genData(7)
  set(K.rec, recs); set(K.meds, meds); set(K.checkin, ck); set(K.skip, sk)
  set(K.family, [{ id: 'f1', name: '小明', relation: '儿子', boundAt: Date.now() - 6 * 864e5 }, { id: 'f2', name: '小红', relation: '女儿', boundAt: Date.now() - 3 * 864e5 }])
  const now = Date.now()
  set(K.visits, [
    { id: 'v1', date: dAfter(5), hospital: '市第一人民医院', department: '心内科', note: '带上近一个月的记录', remindTime: VISIT_REMIND_DEFAULT, status: 'upcoming', createdAt: now, updatedAt: now },
    { id: 'v0', date: dAfter(-25), hospital: '市第一人民医院', department: '心内科', note: '', remindTime: VISIT_REMIND_DEFAULT, status: 'past', createdAt: now - 40 * 864e5, updatedAt: now - 24 * 864e5 }])
  set(K.fpatients, [genPatient('p1', '爸爸', '', 11, 3), genPatient('p2', '妈妈', '', 23, null)])
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
  const f = ymd(start), t = ymd(end); let planned = 0, taken = 0, makeup = 0
  // 应服 = 范围内已到时间的计划时段；已服含补记（PRD 5.6 / 5.8）
  const back = Math.round((new Date(n.getFullYear(), n.getMonth(), n.getDate()) - start) / 864e5) + 1
  const slots = computeSlots(get(K.meds), get(K.checkin), back, Date.now(), get(K.skip)).filter(x => x.date >= f && x.date <= t && x.status !== 'future')
  const meds = live(get(K.meds)).map(m => {
    const mine = slots.filter(x => x.medId === m.id), p = mine.length, tk = mine.filter(x => x.status === 'taken').length
    return { name: m.name, dose: m.dose || '', times: m.times || [], planned: p, taken: tk, rate: p ? r1(tk * 100 / p) : null }
  })
  planned = slots.length; taken = slots.filter(x => x.status === 'taken').length; makeup = slots.filter(x => x.makeup).length
  return { from: f, to: t, days: ndays, generatedAt: Date.now(), average: fin(all), morning: fin(am), evening: fin(pm), count: all.n, series, meds,
    adherence: { planned, taken, makeup, rate: planned ? Math.round(taken * 100 / planned) : null },
    symptoms: Object.entries(sym).map(([name, count]) => ({ name, count })).sort((a, b) => b.count - a.count), footer: REPORT_FOOTER }
}
// 报告卡上的姓名只存本地，不发给后端
export const getReportName = () => uni.getStorageSync('report_name') || ''
export const setReportName = v => uni.setStorageSync('report_name', v)
