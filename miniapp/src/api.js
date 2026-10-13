// 数据层：全部请求 Go 后端（见仓库 server/README.md）；本地只存登录 token、授权记录、昵称、身份等少量状态。
// 后端地址：VITE_API_BASE（默认 http://127.0.0.1:8080）
const ENV = import.meta.env || {}
const BASE = (ENV.VITE_API_BASE || 'http://127.0.0.1:8080').replace(/\/$/, '')
// 订阅消息模板 ID（公众平台申请后填入，与后端 WX_TPL_* 对应）
export const SUBSCRIBE_TMPL_IDS = []
const K = { consent: 'health_consent', invite: 'family_invite', profile: 'profile', role: 'role' }
export const CONSENT_VERSION = '1.0'
const get = (k, d = []) => { const v = uni.getStorageSync(k); return v === '' || v == null ? d : v }
const set = (k, v) => uni.setStorageSync(k, v)
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
export const isLoggedIn = () => !!uni.getStorageSync('token')
// 写操作前调用：未登录则打开登录页，浏览不拦截
export function guard() {
  if (isLoggedIn()) return true
  uni.navigateTo({ url: '/pages/login/login' })
  return false
}
// 昵称只在本机已有时带上；不再每次登录用默认名覆盖后端昵称
export async function login(nickname = (get(K.profile, null) || {}).nickname || '') {
  const consent = { agreedAt: Date.now(), version: CONSENT_VERSION }
  let code
  try { code = (await uni.login({ provider: 'weixin' })).code } catch (e) { uni.showToast({ title: '微信登录失败，请稍后再试', icon: 'none' }); throw e }
  const r = await req('/api/auth/login', 'POST', { code, nickname, consent })
  set('token', r.token)
  set(K.consent, consent)
  if (!get(K.profile, null)) set(K.profile, { nickname: nickname || '用户', since: Date.now() })
  await syncProfile() // 取回后端保存的昵称/头像/身份，重新登录不用再选身份
}
export const getProfile = () => get(K.profile, { nickname: '用户', since: Date.now() })
export const setProfile = p => set(K.profile, { ...getProfile(), ...p })
// 后端返回的 /uploads/... 相对地址补上 API 域名
export const avatarSrc = u => !u ? '' : /^\//.test(u) ? BASE + u : u
const keepMe = u => { if (u) { setProfile({ nickname: u.nickname || getProfile().nickname, avatar: avatarSrc(u.avatarUrl) }); uni.setStorageSync(K.role, u.role || '') } return getProfile() }
// GET /api/me → { nickname, avatarUrl, ... }
export async function syncProfile() { return keepMe(await req('/api/me').catch(() => null)) }
// PUT /api/me { nickname?, avatarUrl? }（昵称 1-20 字）
export async function updateMe(b) { return keepMe(await req('/api/me', 'PUT', b)) }
// POST /api/me/avatar multipart file → { avatarUrl }，再写回资料
export function uploadAvatar(path) {
  return new Promise((ok, no) => uni.uploadFile({ url: BASE + '/api/me/avatar', filePath: path, name: 'file',
    header: { Authorization: 'Bearer ' + (uni.getStorageSync('token') || '') },
    success: r => { let b = {}; try { b = JSON.parse(r.data) } catch (e) {} if (b.code === 0 && b.data && b.data.avatarUrl) ok(b.data.avatarUrl); else { uni.showToast({ title: b.message || '头像上传失败', icon: 'none' }); no(b) } },
    fail: e => { uni.showToast({ title: '网络异常，请稍后再试', icon: 'none' }); no(e) } }))
}

// ---------- 血压 ----------
export async function listRecords() { return (await req('/api/bp')).map(toRec) }
export async function getRecord(id) { return (await listRecords()).find(r => r.id === id) }
export async function addRecord(r) { const d = await req('/api/bp', 'POST', recBody(r)); return { ...d, record: d.record && toRec(d.record) } }
export async function updateRecord(id, r) { return req('/api/bp/' + id, 'PUT', recBody(r)) }
export async function deleteRecord(id) { return req('/api/bp/' + id, 'DELETE') }
// 校验只拦明显输错，措辞中性
export function validate({ sys, dia, hr }) {
  if (!sys || sys < 40 || sys > 300) return '收缩压请填写 40–300 之间的数字'
  if (!dia || dia < 20 || dia > 200) return '舒张压请填写 20–200 之间的数字'
  if (dia >= sys) return '舒张压应小于收缩压，请再核对一下'
  if (hr && (hr < 20 || hr > 250)) return '心率请填写 20–250 之间的数字，或留空'
  return ''
}

// ---------- 用药 ----------
// 时段状态：taken 已服 / skipped 未服（主动点了没吃） / missed 漏服 / pending 待服（窗口内） / future 未到时间
export const MAKEUP_DAYS = 7 // 可补记天数（含今天），与后端一致
export async function listMedDefs() { return req('/api/meds') }
export async function saveMed(m) { const b = { name: m.name, dose: m.dose || '', times: [...m.times].sort() }; return m.id ? req('/api/meds/' + m.id, 'PUT', b) : req('/api/meds', 'POST', b) }
export async function deleteMed(id) { return req('/api/meds/' + id, 'DELETE') }
const fromSlot = s => ({ ...s, status: s.status === 'upcoming' ? 'future' : s.status, sched: ms(s.scheduledAt), checkedAt: s.checkedAt ? ms(s.checkedAt) : null, recordedAt: s.recordedAt ? ms(s.recordedAt) : null, makeup: !!s.isMakeup, canMakeup: !!s.canMakeup, canUndo: !!s.canRevoke, canSkip: !!s.canSkip, canUnskip: !!s.canUnskip })
const RO = s => ({ ...s, canMakeup: false, canUndo: false, canSkip: false, canUnskip: false })
const slotQ = s => '?date=' + s.date + '&time=' + encodeURIComponent(s.time)
// 时段列表（含状态、补记标签），默认最近 7 天
export async function listSlots(days = MAKEUP_DAYS) { return ((await req('/api/meds/history?days=' + days)) || []).map(fromSlot) }
export async function makeupCount() { return ((await req('/api/meds/makeup')) || {}).count || 0 }
// 补记：只能补记最近 7 天内漏服或未服的计划时段（不记录实际服药时间）
export async function makeupCheckin(s) { return req('/api/meds/' + s.medId + '/makeup', 'POST', { date: s.date, time: s.time }) }
export async function undoMakeup(s) { return req('/api/meds/' + s.medId + '/makeup' + slotQ(s), 'DELETE') }
// 没吃（未服）：只能标已到点、未打卡的时段（待服或漏服），7 天内；可撤销。不触发家人漏服通知
export async function skipSlot(s) { return req('/api/meds/' + s.medId + '/skip', 'POST', { date: s.date, time: s.time }) }
export async function unskipSlot(s) { return req('/api/meds/' + s.medId + '/skip' + slotQ(s), 'DELETE') }
// 今日用药：每种药按时间点拆成一行 { id, name, dose, time, done, status, makeup }
export async function listMeds() { const td = ymd(new Date()); return (await listSlots(1)).filter(s => s.date === td && !s.deleted).map(s => ({ ...s, id: s.medId, done: s.status === 'taken' })).sort((a, b) => a.time.localeCompare(b.time)) }
export async function checkin(medId, time) { return req('/api/meds/' + medId + '/checkin', 'POST', { time }) }
export async function listCheckins() {
  const [cks, meds] = await Promise.all([req('/api/meds/checkins'), req('/api/meds')])
  return cks.map(c => ({ medId: c.medId, time: c.slot, at: ms(c.at), makeup: !!c.isMakeup, name: (meds.find(m => m.id === c.medId) || {}).name || '已删除的药品' })).sort((a, b) => b.at - a.at)
}

// ---------- 复诊日提醒（PRD 5.9） ----------
export const VISIT_TMPL_ID = 'zCWcykb6X2AUurA2mKEJDTVH7FabHX8R2lr73aVbWZ4' // 订阅消息模板「复诊提醒」visit_remind，审核通过后填入
export const VISIT_REMIND_DEFAULT = '19:00' // 复诊前一天提醒时间【可调整】
const parseYmd = s => { const [y, m, d] = s.split('-').map(Number); return new Date(y, m - 1, d) }
export const visitRemindAt = v => { const d = parseYmd(v.date); const [h, m] = (v.remindTime || VISIT_REMIND_DEFAULT).split(':').map(Number); return new Date(d.getFullYear(), d.getMonth(), d.getDate() - 1, h, m).getTime() }
export const daysUntil = date => Math.round((parseYmd(date) - new Date(new Date().setHours(0, 0, 0, 0))) / 864e5)
const toVisit = v => v && { ...v, date: v.date || v.visitDate, remindTime: v.remindTime || VISIT_REMIND_DEFAULT }
const ymdOf = s => typeof s === 'string' ? s.slice(0, 10) : s
// GET /api/visits → { upcoming: Visit|null, history: Visit[] }；Visit { id, visitDate, hospital, department, note, remindTime, remindAt, status, pushStatus, todoId? }
export async function listVisits() { const d = (await req('/api/visits')) || {}; return { upcoming: toVisit(d.upcoming || null), history: (d.history || []).map(toVisit) } }
// 即将到来的复诊：status=upcoming 且日期 ≥ 今天
export async function getVisit() { const v = (await listVisits()).upcoming; return v && v.date && v.date.slice(0, 10) >= ymd(new Date()) ? { ...v, date: v.date.slice(0, 10) } : null }
export async function listVisitHistory() { return (await listVisits()).history }
// 保存复诊：v.id 存在则修改；否则新增。已有即将到来的复诊且未传 replace 时抛出 { conflict: 原复诊 }；replace=true 时原记录改为「已取消」进入历史
export async function saveVisit(v, replace = false) {
  const rb = { visitDate: v.date, hospital: (v.hospital || '').slice(0, 30), department: (v.department || '').slice(0, 20), note: (v.note || '').slice(0, 100), remindTime: v.remindTime || VISIT_REMIND_DEFAULT }
  // #ifdef MP-WEIXIN
  if (VISIT_TMPL_ID) await new Promise(ok => uni.requestSubscribeMessage({ tmplIds: [VISIT_TMPL_ID], success: r => { if (r[VISIT_TMPL_ID] === 'accept') req('/api/subscribe', 'POST', { accepted: ['visit'] }).catch(() => {}) }, complete: ok }))
  // #endif
  if (v.id) return req('/api/visits/' + v.id, 'PUT', rb)
  try { return await req('/api/visits', 'POST', { ...rb, replace }, [409]) } catch (e) { if (e && e.code === 409) throw { conflict: toVisit(e.data) || {} }; throw e }
}
export async function cancelVisit(id) { return req('/api/visits/' + id + '/cancel', 'POST') }
export async function deleteVisit(id) { return req('/api/visits/' + id, 'DELETE') }

// ---------- 测量提醒 ----------
// 后端：enabled 总开关 + morningEnabled / eveningEnabled 分开关（未返回时视为开）

// ---------- 家人共享 ----------
export async function createInvite() { const d = await req('/api/family/invite', 'POST'); const inv = { code: d.code, expireAt: ms(d.expiresAt) }; set(K.invite, inv); return inv }
export const getInvite = () => { const v = get(K.invite, null); return v && v.expireAt > Date.now() ? v : null }
// members：能看我的人（id 为绑定关系 id，仅用于解绑）
export async function listFamily() { return ((await req('/api/family')).members || []).map(f => ({ id: f.id, memberId: f.memberId, name: f.nickname || '家人', avatar: avatarSrc(f.avatarUrl), relation: '', boundAt: ms(f.createdAt) })) }
// patients：我能看的人；查看用 patientId
export async function listPatients() { return ((await req('/api/family')).patients || []).map(p => ({ id: p.id, patientId: p.patientId, name: p.nickname || '家人', avatar: avatarSrc(p.avatarUrl), boundAt: ms(p.createdAt) })) }
const noNote = r => ({ ...r, note: '' }) // 家人端不展示备注文字
export async function listPatientRecords(pid) { return (await req('/api/family/patients/' + pid + '/bp')).map(toRec).map(noNote) }
export async function patientStats(pid, range = 'week') { return req('/api/family/patients/' + pid + '/bp/stats?range=' + range) }
// 家人只读：服药历史（含补记标签，无补记/撤销能力）
export async function listPatientSlots(pid, days = MAKEUP_DAYS) { return ((await req('/api/family/patients/' + pid + '/meds/history?days=' + days)) || []).map(fromSlot).map(RO) }
// 家人只读：仅即将到来的复诊日期、医院、科室（不含备注与历史）
export async function getPatientVisit(pid) {
  const v = toVisit(await req('/api/family/patients/' + pid + '/visit'))
  return v && v.date ? { date: ymdOf(v.date), hospital: v.hospital || '', department: v.department || '' } : null
}
// 家人站内消息：漏服/漏测通知，补记后追加「已于……补记」（不撤回、不另发）
export async function listFamilyMessages(pid) { return ((await req('/api/family/messages')) || []).filter(m => !pid || m.patientId === pid).map(m => ({ ...m, createdAt: ms(m.createdAt), makeupNotes: m.makeupNotes || [] })) }
export async function bindFamily(code) { return req('/api/family/bind', 'POST', { code }) }
export async function unfollowPatient(id) { return req('/api/family/' + id, 'DELETE') }
// ---------- 身份入口：patient 我是患者 / family 我是家人 ----------
export const getRole = () => uni.getStorageSync(K.role) || ''
// 身份同时存本机和后端（PUT /api/me {role}），换设备/清缓存后登录仍能取回
export const setRole = r => { uni.setStorageSync(K.role, r); req('/api/me', 'PUT', { role: r }, [undefined, 400, 404]).catch(() => {}) }
export const homeOf = r => r === 'family' ? '/pages/family/home' : '/pages/index/index'
export async function listTodos() { return req('/api/todos') }
export async function doneTodo(id) { return req('/api/todos/' + id + '/done', 'POST') }
export async function unbindFamily(id) { return req('/api/family/' + id, 'DELETE') }

// ---------- 账号 ----------
export async function exportData() { return req('/api/account/export') }
export async function deleteAccount() {
  await req('/api/account', 'DELETE')
  uni.clearStorageSync()
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
  const q = opt.from && opt.to ? `from=${opt.from}&to=${opt.to}` : 'days=' + (opt.days || 30)
  const r = await req('/api/report?' + q)
  return { ...r, generatedAt: ms(r.generatedAt), footer: r.footer || REPORT_FOOTER }
}
// 报告卡上的姓名只存本地，不发给后端
export const getReportName = () => uni.getStorageSync('report_name') || ''
export const setReportName = v => uni.setStorageSync('report_name', v)
