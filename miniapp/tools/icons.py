import os,subprocess
P={
'home':'<path d="M3.5 10.5L12 4l8.5 6.5V19a1.5 1.5 0 01-1.5 1.5h-4v-5.5h-6v5.5H5A1.5 1.5 0 013.5 19z"/>',
'chart':'<path d="M4 20V4M4 20h16"/><path d="M8 15l3.5-4 3 2.5L20 7"/>',
'user':'<circle cx="12" cy="8.5" r="3.8"/><path d="M4.5 20.5c1-3.8 3.9-5.8 7.5-5.8s6.5 2 7.5 5.8"/>',
'pill':'<rect x="2.8" y="8.6" width="18.4" height="6.8" rx="3.4" transform="rotate(-45 12 12)"/><path d="M9.6 9.6l4.8 4.8"/>',
'family':'<circle cx="9" cy="8" r="3.3"/><circle cx="17" cy="9.5" r="2.4"/><path d="M2.8 19.5c.8-3.3 3.2-5 6.2-5s5.4 1.7 6.2 5M15.5 14.3c2.8 0 4.8 1.5 5.7 4.2"/>',
'bell':'<path d="M6 16V11a6 6 0 0112 0v5l1.5 2h-15z"/><path d="M10 20.5a2 2 0 004 0"/>',
'check':'<path d="M5 12.5l4.5 4.5L19 7.5"/>',
'plus':'<path d="M12 5v14M5 12h14"/>',
'right':'<path d="M9 6l6 6-6 6"/>',
'clock':'<circle cx="12" cy="12" r="8.5"/><path d="M12 7.5V12l3 2"/>',
'sun':'<circle cx="12" cy="12" r="4"/><path d="M12 3v2M12 19v2M3 12h2M19 12h2M5.6 5.6l1.4 1.4M17 17l1.4 1.4M5.6 18.4L7 17M17 7l1.4-1.4"/>',
'moon':'<path d="M19 14.5A7.5 7.5 0 019.5 5a7.5 7.5 0 109.5 9.5z"/>',
'del':'<path d="M9 6h11v12H9l-5-6z"/><path d="M12.5 9.5l5 5M17.5 9.5l-5 5"/>',
'shield':'<path d="M12 3l7.5 3v5.5c0 4.6-3.2 8.2-7.5 9.5-4.3-1.3-7.5-4.9-7.5-9.5V6z"/><path d="M8.8 12l2.3 2.3 4.3-4.3"/>',
'download':'<path d="M12 4v11M7.5 10.5L12 15l4.5-4.5M5 19.5h14"/>',
'trash':'<path d="M4.5 7h15M9.5 7V4.5h5V7M6.5 7l1 13h9l1-13M10 11v5.5M14 11v5.5"/>',
'heart':'<path d="M12 20s-7.5-4.6-7.5-10.3A4.2 4.2 0 0112 7.2a4.2 4.2 0 017.5 2.5C19.5 15.4 12 20 12 20z"/><path d="M7 12.5h3l1.5-3 2 5 1.5-2H17"/>',
'eye':'<path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12z"/><circle cx="12" cy="12" r="3"/>',
'refresh':'<path d="M19.5 12a7.5 7.5 0 11-2.2-5.3M19.5 4.5v4h-4"/>',
'edit':'<path d="M4 20h4L19 9l-4-4L4 16z"/><path d="M13.5 6.5l4 4"/>',
}
C={'g':'#10C495','w':'#FFFFFF','o':'#FBA900','b':'#37A5E6','k':'#000000','9':'#999999','6':'#666666','l':'#31CD4C','r':'#E5484D'}
d=os.path.dirname(os.path.abspath(__file__))+'/../src/static/'
def svg(n,c,sw=1.8):return f'<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="{c}" stroke-width="{sw}" stroke-linecap="round" stroke-linejoin="round">{P[n]}</svg>'
for n in P:
  for k,c in C.items(): open(f'{d}icons/{n}-{k}.svg','w').write(svg(n,c,2.4 if n in('check','plus') else 1.8))
html='<html><body style="margin:0;background:transparent">'
for n,t in(('home','index'),('chart','record'),('user','mine')):
  for s,c in(('','#999999'),('-on','#10C495')):
    open(f'/tmp/t.html','w').write(f'<html><body style="margin:0;background:transparent">'+svg(n,c,2 if s else 1.8).replace('width="48" height="48"','width="81" height="81"')+'</body></html>')
    subprocess.run(['google-chrome','--headless','--no-sandbox','--disable-gpu','--hide-scrollbars','--default-background-color=00000000','--window-size=81,81',f'--screenshot={d}tab/{t}{s}.png','file:///tmp/t.html'],capture_output=True)
