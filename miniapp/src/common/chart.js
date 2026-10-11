import { WEEK, dayKey } from '../api.js'
// 生成折线图 SVG（data URI），H5 与小程序 <image> 均可显示。只画数据，不画任何参考范围线。
export function buildChart(records, days, W = 311, H = 170) {
  const now = new Date(); const start = days ? new Date(now.getFullYear(), now.getMonth(), now.getDate() - days + 1).getTime() : 0
  const byDay = {}
  records.filter(r => r.time >= start).forEach(r => { const k = dayKey(r.time); (byDay[k] = byDay[k] || { t: r.time, s: [], d: [] }); byDay[k].s.push(r.sys); byDay[k].d.push(r.dia); byDay[k].t = Math.min(byDay[k].t, r.time) })
  const pts = Object.values(byDay).sort((a, b) => a.t - b.t).map(x => ({ t: x.t, s: avg(x.s), d: avg(x.d) }))
  if (!pts.length) return { src: '', pts }
  const all = pts.flatMap(p => [p.s, p.d]); let lo = Math.floor((Math.min(...all) - 8) / 10) * 10, hi = Math.ceil((Math.max(...all) + 8) / 10) * 10
  const step = Math.max(10, Math.ceil((hi - lo) / 3 / 10) * 10); hi = lo + step * 3
  const L = 26, R = W - 8, T = 10, B = H - 20
  const x = i => pts.length === 1 ? (L + R) / 2 : L + (R - L) * i / (pts.length - 1)
  const y = v => B - (v - lo) / (hi - lo) * (B - T)
  let g = ''
  for (let i = 0; i <= 3; i++) { const v = lo + step * i, yy = y(v).toFixed(1); g += `<line x1="${L}" x2="${W}" y1="${yy}" y2="${yy}" stroke="#F0F2F1"/><text x="0" y="${+yy + 3}" font-size="10" fill="#999" font-family="Arial">${v}</text>` }
  const line = k => pts.map((p, i) => (i ? 'L' : 'M') + x(i).toFixed(1) + ',' + y(p[k]).toFixed(1)).join(' ')
  const area = k => line(k) + ` L${x(pts.length - 1).toFixed(1)},${B} L${x(0).toFixed(1)},${B} Z`
  const dots = (k, c) => pts.length > 16 ? `<circle cx="${x(pts.length - 1)}" cy="${y(pts[pts.length - 1][k])}" r="3.5" fill="#fff" stroke="${c}" stroke-width="2.2"/>` : pts.map((p, i) => `<circle cx="${x(i).toFixed(1)}" cy="${y(p[k]).toFixed(1)}" r="${i === pts.length - 1 ? 4 : 2.5}" fill="#fff" stroke="${c}" stroke-width="${i === pts.length - 1 ? 2.5 : 1.8}"/>`).join('')
  const every = Math.ceil(pts.length / 7)
  const labels = pts.map((p, i) => { if ((pts.length - 1 - i) % every) return ''; const d = new Date(p.t); const last = i === pts.length - 1; const t = days === 7 ? WEEK[d.getDay()] : `${d.getMonth() + 1}/${d.getDate()}`; return `<text x="${x(i).toFixed(1)}" y="${H - 4}" font-size="10" fill="${last ? '#10C495' : '#999'}" text-anchor="middle" font-family="Arial">${t}</text>` }).join('')
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${W * 2}" height="${H * 2}" viewBox="0 0 ${W} ${H}"><defs><linearGradient id="g1" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#10C495" stop-opacity=".22"/><stop offset="1" stop-color="#10C495" stop-opacity="0"/></linearGradient><linearGradient id="g2" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#37A5E6" stop-opacity=".16"/><stop offset="1" stop-color="#37A5E6" stop-opacity="0"/></linearGradient></defs>${g}<path d="${area('s')}" fill="url(#g1)"/><path d="${area('d')}" fill="url(#g2)"/><path d="${line('s')}" fill="none" stroke="#10C495" stroke-width="2.2" stroke-linejoin="round"/><path d="${line('d')}" fill="none" stroke="#37A5E6" stroke-width="2.2" stroke-linejoin="round"/>${dots('s', '#10C495')}${dots('d', '#37A5E6')}${labels}</svg>`
  return { src: 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg), pts }
}
export const avg = a => a.length ? Math.round(a.reduce((x, y) => x + y, 0) / a.length) : 0
export function sessionAvg(records) {
  const m = records.filter(r => new Date(r.time).getHours() < 12), e = records.filter(r => new Date(r.time).getHours() >= 12)
  const f = l => l.length ? avg(l.map(r => r.sys)) + '/' + avg(l.map(r => r.dia)) : '--'
  return { morning: f(m), evening: f(e), mn: m.length, en: e.length }
}
