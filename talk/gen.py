#!/usr/bin/env python3
# Rebuilds the grouped slides of the deck from base/slides plus the canvas scenes in scenes/.
# cover.html, why.html, demo-video.html and wrap.html are not generated: edit them directly in deck/project/slides.
import re, json, os
S = os.path.dirname(os.path.abspath(__file__))
BK = S + '/base/slides'            # slides as they were before the sidebar/split pass; gen.py reuses their content
SL = S + '/deck/project/slides'    # generated output; publish with root = S + '/deck'
SC = S + '/scenes'                 # canvas scene bodies (plain JS, no '<' or '&')

def rd(p): return open(p, encoding='utf-8').read()
def wr(name, text): open(f'{SL}/{name}.html', 'w', encoding='utf-8').write(text)
def bk(name): return rd(f'{BK}/{name}.html')
def emb(name): return re.search(r'<x-embed[^>]*>(.*?)</x-embed>', bk(name), re.S).group(1)
def old_aside(name): return re.search(r'<aside>(.*?)</aside>', bk(name), re.S).group(1)
def code_p(name): return re.search(r"(<p style=\"font-family:'JetBrains Mono'[^>]*white-space:nowrap\">.*?</p>)", bk(name), re.S).group(1)

PRE = '''<style>html,body{margin:0;height:100%;overflow:hidden;background:transparent}canvas{display:block;width:100vw;height:100vh;border-radius:16px}</style>
<canvas id="c"></canvas>
<script>
(function(){
var c=document.getElementById('c'),g=c.getContext('2d'),BW=@BW@,BH=@BH@,END=@END@,CLEAR=@CLEAR@;
var F=parseFloat((location.hash.match(/t=([\\d.]+)/)||[])[1]);
function fit(){var d=Math.min(window.devicePixelRatio||1,2);c.width=innerWidth*d;c.height=innerHeight*d;g.setTransform(c.width/BW,0,0,c.height/BH,0,0)}
function cl(v){return Math.max(0,Math.min(1,v))}
function P(t,a,b){return cl((t-a)/(b-a))}
function mix(a,b,k){return a+(b-a)*k}
function eo(k){return 1-Math.pow(1-k,3)}
function eio(k){return k>.5?1-Math.pow(-2*k+2,3)/2:4*k*k*k}
function bk(k){var s=1.70158;return 1+(s+1)*Math.pow(k-1,3)+s*Math.pow(k-1,2)}
function win(t,a,b){return Math.min(P(t,a,a+.35),1-P(t,b-.35,b))}
var CY='#5DC9E2',OR='#F2A65A',PK='#FF7A9A',TX='#E6EDF3',MU='#8B9BAA',BG='#101820',LN='#2A3A4A';
var SA='system-ui,-apple-system,"Segoe UI",Roboto,Arial,sans-serif',MO='ui-monospace,SFMono-Regular,Menlo,Consolas,monospace';
function txt(s,X,Y,f,col,al,alg){if(!(al>0))return;g.globalAlpha=al;g.font=f;g.fillStyle=col;g.textAlign=alg||'left';g.textBaseline='middle';g.fillText(s,X,Y);g.globalAlpha=1}
function rr(X,Y,w,h,r){g.beginPath();g.moveTo(X+r,Y);g.arcTo(X+w,Y,X+w,Y+h,r);g.arcTo(X+w,Y+h,X,Y+h,r);g.arcTo(X,Y+h,X,Y,r);g.arcTo(X,Y,X+w,Y,r);g.closePath()}
function disc(X,Y,r,col,al){if(!(al>0))return;if(!(r>0))return;g.globalAlpha=al;g.beginPath();g.arc(X,Y,r,0,6.2832);g.fillStyle=col;g.fill();g.globalAlpha=1}
function ring(X,Y,r,col,lw,al){if(!(al>0))return;if(!(r>0))return;g.globalAlpha=al;g.beginPath();g.arc(X,Y,r,0,6.2832);g.strokeStyle=col;g.lineWidth=lw;g.stroke();g.globalAlpha=1}
function seg(x1,y1,x2,y2,col,lw,al,dash){if(!(al>0))return;g.globalAlpha=al;if(dash)g.setLineDash([8,10]);g.beginPath();g.moveTo(x1,y1);g.lineTo(x2,y2);g.strokeStyle=col;g.lineWidth=lw;g.lineCap='round';g.stroke();g.setLineDash([]);g.globalAlpha=1}
'''
POST = '''function frame(t){g.clearRect(0,0,BW,BH);if(!CLEAR){g.fillStyle=BG;g.fillRect(0,0,BW,BH)}draw(t);if(!CLEAR){var f=1-P(t,0,.4);if(f>0){g.globalAlpha=f;g.fillStyle=BG;g.fillRect(0,0,BW,BH);g.globalAlpha=1}}}
var T0=-1,LT=0;
function fr(n){if(0>T0)T0=n;if(n-LT>900)T0=n;LT=n;frame(Math.min((n-T0)/1000,END));requestAnimationFrame(fr)}
if(window.IntersectionObserver){new IntersectionObserver(function(e){if(e[0].isIntersecting)T0=-1}).observe(c)}
function go(){fit();if(!isNaN(F))frame(F)}
go();addEventListener('resize',go);if(isNaN(F))requestAnimationFrame(fr);
})();
</script>
'''
def scene(name, bw, bh, end, clear=0, defs=''):
    body = rd(f'{SC}/{name}.js')
    assert '<' not in body and '&' not in body, name
    pre = PRE.replace('@BW@', str(bw)).replace('@BH@', str(bh)).replace('@END@', str(end)).replace('@CLEAR@', str(clear))
    if clear: pre = pre.replace(';border-radius:16px', '')
    return pre + defs + body + POST

LIGHT = "background:#F6F4EE;color:#101820;font-family:'IBM Plex Sans', Arial, sans-serif;padding:112px 128px 112px;display:flex;flex-direction:column;gap:40px"
def head(eyebrow, title):
    return ('<div style="display:flex;flex-direction:column;gap:12px">\n'
            f'<p style="font-size:24px;font-weight:600;letter-spacing:2px;text-transform:uppercase;color:#006E8A">{eyebrow}</p>\n'
            f'<h2 style="font-size:64px;font-weight:600;line-height:1.1">{title}</h2>\n</div>\n')
def mk(sid, eyebrow, title, body, aside):
    return f'<section id="{sid}" style="{LIGHT}">\n' + head(eyebrow, title) + body + f'\n<aside>{aside}</aside>\n</section>\n'
def xe(x, y, w, h, inner, eid=None):
    i = f' id="{eid}"' if eid else ''
    return f'<x-embed{i} style="position:absolute;left:{x}px;top:{y}px;width:{w}px;height:{h}px">{inner}</x-embed>\n'
def rows(items, x, y, w):
    out = f'<div style="position:absolute;left:{x}px;top:{y}px;width:{w}px;display:flex;flex-direction:column">\n'
    for i, (h, p) in enumerate(items):
        bt = '2px solid #101820' if i == 0 else '1px solid #DDD8CC'
        out += (f'<div style="display:flex;flex-direction:column;gap:8px;padding:20px 0px;border-top:{bt}">'
                f'<h3 style="font-size:30px;font-weight:600;line-height:1.2">{h}</h3>'
                f'<p style="font-size:24px;line-height:1.4;color:#3D4852">{p}</p></div>\n')
    return out + '</div>\n'
def cols(items, x, y, w):
    out = f'<div style="position:absolute;left:{x}px;top:{y}px;width:{w}px;display:flex;gap:48px;align-items:start">\n'
    for h, p in items:
        out += (f'<div style="flex:1;display:flex;flex-direction:column;gap:8px">'
                f'<h3 style="font-size:30px;font-weight:600;line-height:1.2">{h}</h3>'
                f'<p style="font-size:24px;line-height:1.4;color:#3D4852">{p}</p></div>\n')
    return out + '</div>\n'
def stmt(text, x, y, w):
    return f'<p style="position:absolute;left:{x}px;top:{y}px;width:{w}px;font-size:40px;line-height:1.3">{text}</p>\n'
def panel(p, w):
    return f'<div style="width:{w}px;background:#101820;border-radius:16px;padding:36px 40px;display:flex;flex-direction:column">\n{p}\n</div>\n'

# ---------------------------------------------------------------- sidebar
GROUPS = {
    'how': ['Code', 'Play', 'CheckCircle'],
    'sdk': ['Settings', 'Warning', 'Database', 'Activity'],
    'go': ['Code', 'Lightning', 'Warning', 'Wrench', 'Lightbulb'],
    'demo': ['Cloud', 'Code', 'Database'],
}
import math
ACX, ACY, AR, AOFF, ASTEP = 1269, 657, 658, 60, 12
GLYPH = [
    '<path d="M-10 -13 L-22 0 L-10 13 M10 -13 L22 0 L10 13 M4 -17 L-4 17"/>',
    '<path d="M5 -21 L-11 3 L1 3 L-5 21 L11 -3 L-1 -3 Z" fill="@C@" stroke="none"/>',
    '<path d="M0 -19 L21 17 L-21 17 Z"/><path d="M0 -6 L0 5"/><circle cx="0" cy="11" r="1.5" fill="@C@" stroke="none"/>',
    '<rect x="-23" y="-18" width="46" height="36" rx="6"/><path d="M-13 -7 L-5 0 L-13 7 M1 8 L12 8"/>',
    '<path d="M-7.65 5.51 A13 13 0 1 1 7.91 5.32"/><path d="M-6 13 L6 13 M-4 20 L4 20"/>',
]
# AWS marks and the Go logo, copied from github.com/glincker/thesvg (public/icons/<name>) into logos/. Trademarks belong to their owners.
LG = S + '/logos'
def logo_inner(name):
    t = rd(f'{LG}/{name}.svg')
    vb = re.search(r'viewBox="0 0 ([\d.]+) ([\d.]+)"', t)
    t = re.sub(r'<\?xml[^>]*\?>|<title>.*?</title>|<!--.*?-->', '', t, flags=re.S)
    t = re.search(r'<svg[^>]*>(.*)</svg>', t, re.S).group(1)
    t = re.sub(r'\s+id="[^"]*"', '', t)
    t = re.sub(r'>\s+<', '><', t).strip()
    return float(vb.group(1)), float(vb.group(2)), t
def mono_aws(name, col):
    # AWS architecture icon reduced to its glyph in one colour: drop the coloured tile, tint the white mark and
    # give it a hairline stroke so it carries the same weight as the deck's own icons.
    w0, h0, inner = logo_inner(name)
    inner = re.sub(r'<g fill="#[0-9A-Fa-f]{6}"><rect[^>]*></rect></g>', '', inner)
    assert 'fill="#FFFFFF"' in inner and '<rect' not in inner, name
    return inner.replace('fill="#FFFFFF"', f'fill="{col}" stroke="{col}" stroke-width="1" stroke-linejoin="round"')
def logo_svg(name, label, x, y, h, col='#006E8A'):
    return (f'<svg id="logo-{name}" aria-label="{label}" style="position:absolute;left:{x}px;top:{y}px" width="{h}" height="{h}" '
            f'viewBox="9 9 62 62">{mono_aws(name, col)}</svg>\n')
def go_logo_g(cx, cy, h, col):
    # The Go logo (single colour in the source), tinted and centred on cx, cy.
    w0, h0, t = logo_inner('go')
    assert 'fill="#fff"' in t
    k = h / h0
    return f'<g transform="translate({cx - w0 * k / 2:.1f} {cy - h / 2:.1f}) scale({k:.4f})">{t.replace('fill="#fff"', f'fill="{col}"')}</g>'
def aws_path(name):
    # The mark of an AWS icon as path data for a canvas Path2D, rounded to one decimal to keep embeds small.
    d = re.search(r'<path[^>]*? d="([^"]*)"', rd(f'{LG}/{name}.svg'), re.S).group(1)
    d = re.sub(r'-?\d+\.\d+', lambda m: ('%.1f' % float(m.group(0))).rstrip('0').rstrip('.'), d)
    d = re.sub(r'\s+', ' ', d)
    return re.sub(r' ?([MLCZHVSQAmlczhvsqa]) ?', r'\1', d).strip()
def code_html(src):
    # Go source for a slide: escaped, indentation kept, comments dimmed, «marked» calls in cyan.
    out = []
    for line in src.strip('\n').split('\n'):
        ind = len(line) - len(line.lstrip(' '))
        t = line.strip().replace('&', '&amp;')
        if '//' in t:
            i = t.index('//'); t = t[:i] + '<span style="color:#8B9BAA">' + t[i:] + '</span>'
        t = t.replace('«', '<span style="color:#5DC9E2">').replace('»', '</span>')
        out.append('&#160;' * ind + t)
    return ("<p style=\"font-family:'JetBrains Mono', 'Courier New', monospace;font-size:24px;line-height:1.5;color:#E6EDF3;white-space:nowrap\">"
            + '<br>'.join(out) + '</p>')
def code_slide(sid, title, src, notes, aside):
    return mk(sid, 'Demo · Code', title, panel(code_html(src), 1160) + rows(notes, 1336, 264, 456), aside)
LOGOS = {'demo-data': ('aws-dynamodb', 'Amazon DynamoDB', 1720, 112, 72)}   # why.html (hand-written) carries aws-lambda at the same spot
# Arc states: [centre x, centre y, radius, icon radius, icon step, angle of first icon, glow angle, glow distance beyond the arc, glow radius]
BIGR = [960, 1100, 1157, 1217, 5.2, -44, -37, 160, 520]
def BIGL(k): return [960, 1100, 1157, 1217, 5.2, round(41 - 5.2 * k, 2), 37, 160, 520]
def SMALL(k): return [430, 430, 430, 485, 9.5, round(45 - 9.5 * k, 2), 45, 110, 330]
def arc_svg(s, foc, oc='#006E8A', show_icons=True, extra=''):
    # A still picture of the arc. It must match what the canvas in gonav.js draws for the same state.
    pt = lambda phi, r: (s[0] - r * math.sin(math.radians(phi)), s[1] - r * math.cos(math.radians(phi)))
    gx, gy = pt(s[6], s[2] + s[7])
    out = ('<svg id="nav-arc-s" aria-label="Topic navigation" style="position:absolute;left:0px;top:0px" width="1920" height="540" viewBox="0 0 1920 540"><defs>'
           f'<radialGradient id="ag" gradientUnits="userSpaceOnUse" cx="{gx:.1f}" cy="{gy:.1f}" r="{s[8]}">'
           '<stop offset="0" stop-color="rgb(0,173,216)" stop-opacity="0.8"/><stop offset="0.32" stop-color="rgb(0,173,216)" stop-opacity="0.42"/>'
           '<stop offset="0.62" stop-color="rgb(0,173,216)" stop-opacity="0.2"/><stop offset="1" stop-color="rgb(0,173,216)" stop-opacity="0"/></radialGradient>')
    icons = ''
    for j in range(5 if show_icons else 0):
        x, y = pt(s[5] + j * s[4], s[3])
        if x < -60 or x > 1980 or y < -60 or y > 600: continue
        f = 1 if j == foc else 0
        if f:
            out += (f'<radialGradient id="af" gradientUnits="userSpaceOnUse" cx="{x:.1f}" cy="{y:.1f}" r="62"><stop offset="0" stop-color="rgb(0,173,216)" stop-opacity="0.9"/>'
                    '<stop offset="0.6" stop-color="rgb(0,173,216)" stop-opacity="0.85"/><stop offset="1" stop-color="rgb(0,173,216)" stop-opacity="0"/></radialGradient>')
            icons += f'<circle cx="{x:.1f}" cy="{y:.1f}" r="62" fill="url(#af)"/>'
        col = '#F6F4EE' if f else oc
        icons += (f'<g transform="translate({x:.1f} {y:.1f}) scale({1.3 if f else 0.8})" opacity="{1 if f else 0.6}" fill="none" stroke="{col}" '
                  f'stroke-width="4.5" stroke-linecap="round" stroke-linejoin="round">{GLYPH[j].replace("@C@", col)}</g>')
    out += (f'</defs><path fill-rule="evenodd" fill="url(#ag)" d="M0 0 H1920 V540 H0 Z M{s[0] - s[2]} {s[1]} a{s[2]} {s[2]} 0 1 0 {2 * s[2]} 0 a{s[2]} {s[2]} 0 1 0 {-2 * s[2]} 0 Z"/>')
    return out + icons + extra + '</svg>\n'
GOLOGO = (1726, 104, 48, '#F6F4EE')   # centre x, centre y, height, colour of the Go logo on the opener's arc
def arc_nav(idx, prev, teal=False, intro=False):
    # Go section. Every slide carries a still SVG of the arc; topic cards also carry a canvas that paints over
    # it and animates: start state -> big arc on this topic -> ring and line -> shrink to the compact arc.
    if teal:                       # section opener: big, resting in the top-right, the Go logo instead of topic icons
        # glow under the animated backdrop (its stripes pass over the glow), logo above it
        w0, h0, _ = logo_inner('go'); h = GOLOGO[2]; w = round(h * w0 / h0)
        logo = (f'<svg id="nav-arc-logo" aria-label="Go" style="position:absolute;left:{GOLOGO[0] - w // 2}px;top:{GOLOGO[1] - h // 2}px" '
                f'width="{w}" height="{h}" viewBox="0 0 {w} {h}">{go_logo_g(w / 2, h / 2, h, GOLOGO[3])}</svg>\n')
        return arc_svg(BIGR, -1, show_icons=False), logo
    if prev is None:               # content slide: compact arc at rest
        return arc_svg(SMALL(idx), idx), ''
    s0, f0 = (BIGR, -1) if prev < 0 else (SMALL(prev), prev)
    defs = f'var S0={s0},S1={BIGL(idx)},S2={SMALL(idx)},F0={f0},F1={idx};\n'
    # first card: the opener shows no topic icons, so the start state has none either; the canvas fades them in
    return arc_svg(s0, f0, show_icons=prev >= 0) + xe(0, 0, 1920, 540, scene('gonav', 1920, 540, 100000, 1, defs), 'nav-arc'), ''
def nav(group, idx, dark=False, intro=False, teal=False):
    rgb, chip, on, off =('24,35,46', '#5DC9E2', '#101820', '#F6F4EE') if dark else ('236,232,221', '#006E8A', '#F6F4EE', '#006E8A')
    fadeo = [1, 0.3, 0.13, 0.06, 0.03]
    out = (f'<div id="nav-strip" style="position:absolute;left:1816px;top:0px;width:104px;height:1080px;'
           f'background:linear-gradient(to bottom, rgba({rgb},0) 0%, rgba({rgb},1) 34%, rgba({rgb},1) 66%, rgba({rgb},0) 100%)"></div>\n')
    if intro:
        out += xe(1380, 440, 540, 200, scene('navline', 540, 200, 100000, 1), 'nav-line')
    for j, name in enumerate(GROUPS[group]):
        top = 504 + (j - idx) * 112
        if top < -60 or top > 1070: continue
        col = chip if j == idx else off
        op = fadeo[min(abs(j - idx), 4)]
        out += f'<x-icon id="nav-{group}-{j}" name="{name}" style="position:absolute;left:1832px;top:{top}px;width:72px;height:72px;color:{col};opacity:{op}"></x-icon>\n'
    return out
NAVRE = re.compile(r'<(div|x-icon|x-embed|svg) id="nav-[^"]*"[^>]*>.*?</\1>\n?', re.S)
def finish(text, group, idx, trans, dark=False, intro=False, teal=False, prev=None):
    text = NAVRE.sub('', text)
    if group == 'go':
        back, front = arc_nav(idx, prev, teal, intro)
        if teal:      # section opener: glow, then the transparent animated backdrop, then the logo, all under the text
            assert '</x-embed>\n<p ' in text
            text = text.replace('<x-embed', back + '<x-embed', 1).replace('</x-embed>\n', '</x-embed>\n' + front, 1)
        else:
            text = text.replace('<aside>', front + '<aside>', 1)
            text = re.sub(r'(<section id="[^"]+"[^>]*>\n)', lambda m: m.group(1) + back, text, count=1)
    elif group: text = text.replace('<aside>', nav(group, idx, dark, intro, teal) + '<aside>', 1)
    def fix(m):
        tag = re.sub(r' data-transition="[^"]*"', '', m.group(0))
        return tag.replace('<section id="%s"' % m.group(1), '<section id="%s" data-transition="%s"' % (m.group(1), trans), 1) if trans else tag
    return re.sub(r'<section id="([^"]+)"[^>]*>', fix, text, count=1)

# Topic cards carry no "N of M" line: the sidebar (SDK) and the corner arc (Go) show the position.
def intro_go(sid, title, line, aside):
    body = (f'<h2 style="position:absolute;left:560px;top:470px;width:1232px;font-size:120px;font-weight:600;line-height:1.1;letter-spacing:-2px">{title}</h2>\n'
            f'<p style="position:absolute;left:560px;top:624px;width:1232px;font-size:30px;line-height:1.4;color:#3D4852">{line}</p>\n')
    return f'<section id="{sid}" style="{LIGHT}">\n{body}<aside>{aside}</aside>\n</section>\n'
def intro(sid, title, line, aside):
    body = (f'<h2 style="position:absolute;left:128px;top:470px;width:1220px;text-align:right;font-size:120px;font-weight:600;line-height:1.1;letter-spacing:-2px">{title}</h2>\n'
            f'<p style="position:absolute;left:128px;top:624px;width:1220px;text-align:right;font-size:30px;line-height:1.4;color:#3D4852">{line}</p>\n')
    return f'<section id="{sid}" style="{LIGHT}">\n{body}<aside>{aside}</aside>\n</section>\n'

# ---------------------------------------------------------------- slides
NEW = {}
NEW['sdk-ops'] = intro('sdk-ops', 'Operations', 'Eight building blocks, three of which suspend the function.', 'Topic card, two seconds. The sidebar on the right tracks the four SDK topics: operations, error handling, state, observability. Say: first, what you can actually call.')
NEW['sdk-errors'] = intro('sdk-errors', 'Error handling', 'Retries, backoff, and what replay does with a failure.', 'Topic card, two seconds. Say: second, what happens when something fails.')
NEW['sdk-state'] = intro('sdk-state', 'State', 'What goes into a checkpoint, and how big it can be.', 'Topic card, two seconds. Say: third, where your data lives between invocations.')
NEW['sdk-obs'] = intro('sdk-obs', 'Observability', 'Seeing one execution across many invocations.', 'Topic card, two seconds. Say: last, how you see what an execution did.')
NEW['go-i-shape'] = intro_go('go-i-shape', 'API shape', 'Operations are generic functions, not methods.', 'Topic card, two seconds. The sidebar now tracks the five Go topics.')
NEW['go-i-conc'] = intro_go('go-i-conc', 'Concurrency', 'Why a durable context belongs to one goroutine.', 'Topic card, two seconds. Say: this is the most Go-specific part of the talk.')
NEW['go-i-errors'] = intro_go('go-i-errors', 'Errors', 'Errors are values, and one of them is a signal.', 'Topic card, two seconds.')
NEW['go-i-tools'] = intro_go('go-i-tools', 'Tooling', 'A linter that knows the rules, and a local test runner.', 'Topic card, two seconds.')
NEW['go-i-diff'] = intro_go('go-i-diff', 'Surprises', 'Sometimes it doesn\'t <b><span style="color:#00ADD8">GO</span></b> as you think it should.', 'Topic card, two seconds.')

NEW['errors'] = mk('errors', 'The SDK · Error handling', 'Where an error happens decides what happens',
    xe(128, 264, 1664, 560, scene('instep', 1664, 560, 10.5)) +
    stmt('Wrap anything error-prone in a step. Code outside a step gets no retries.', 128, 856, 1664),
    'No clicks; the panel plays once, about ten seconds. Top lane, inside a step: the attempt fails, the runner goes back to the start of the step, fails again, and succeeds on the third attempt; the handler continues. Each attempt is checkpointed and the invocation ends between attempts, so the backoff uses no compute. Bottom lane, outside a step: the same error between operations fails the whole execution at once, with no retry. When retries run out, the final error is checkpointed and handed to the handler, which can catch it and carry on.')
NEW['errors-2'] = mk('errors-2', 'The SDK · Error handling', 'Retries back off without using compute',
    xe(128, 264, 1218, 688, emb('errors')) +
    rows([('The default', 'durable.ExponentialBackoff(): six attempts, 5 s doubling to a 60 s cap, full jitter.'),
          ('Other presets', 'durable.NoRetry(), durable.MustLinearBackoff(...), or your own function.'),
          ('On replay', 'The same error comes back at the same point. The body does not run again.')], 1394, 264, 398),
    old_aside('errors'))
NEW['errors-3'] = mk('errors-3', 'The SDK · Error handling', 'Step semantics: at least once, or at most once',
    xe(128, 264, 1664, 560, scene('semantics', 1664, 560, 11.5)) +
    stmt('At least once is the default. Choose at most once for side effects you cannot repeat, such as a payment.', 128, 856, 1664),
    'No clicks; the panel plays once, about eleven seconds. Both lanes show the same accident: Lambda is interrupted after the step body ran but before its result was checkpointed. Top lane, AtLeastOncePerRetry, the default: on replay the body runs again, so the charge happens twice. That is why steps should be idempotent. Bottom lane, AtMostOncePerRetry: on replay the body is not run again; the interrupted attempt counts as failed, and when no retries remain the step returns a StepError with ErrorType StepInterruptedError. You do not know whether the charge went through, so check the payment system. Set it with durable.WithSemantics(durable.AtMostOncePerRetry). The charge counter is illustrative.')
NEW['state'] = mk('state', 'The SDK · State', 'State is what you serialize',
    xe(128, 264, 1664, 520, scene('roundtrip', 1664, 520, 8.5)) +
    cols([('JSON by default', 'Results are stored with encoding/json. Replay decodes the stored result and skips the code that produced it.'),
          ('Struct rules apply', 'Tags and unexported fields behave exactly as they do in encoding/json.')], 128, 820, 1664),
    'No clicks; the panel plays once, about eight seconds. A Go value is marshaled into the checkpoint: the two exported fields make it into the JSON, the unexported one does not. On replay the value is unmarshaled again and the unexported field comes back empty. There is no state store API: state is the set of operation results in the checkpoint log, so what you return from a step is what you can rely on later. The default is durable.JSONSerdes, encoding/json with HTML escaping disabled. The SDK applies this round trip on the first run too, so a value that does not survive JSON shows up immediately, not as a surprise during replay. The Order type is an illustration.')
NEW['state-2'] = mk('state-2', 'The SDK · State', 'Limits, and what to do about them',
    xe(128, 264, 600, 688, emb('state')) +
    rows([('256 KB per step result', '262,144 bytes per checkpoint. The local test runner enforces the same limit.'),
          ('1 MB per Invoke input', '1,048,576 bytes.'),
          ('Large payloads', 'NewFileSystemSerdes writes the value to EFS or S3 Files and checkpoints a reference. Never point it at /tmp.'),
          ('Custom Serdes', 'Marshal and Unmarshal, set with WithSerdes or WithStepSerdes, for encryption or compression.')], 776, 264, 1016),
    old_aside('state'))
NEW['observability'] = mk('observability', 'The SDK · Observability', 'One execution, many invocations',
    xe(128, 264, 752, 688, emb('observability')) +
    stmt('Lambda writes one log stream per invocation, so the history of a single execution is scattered across streams.', 944, 420, 848),
    'No clicks; the panel plays once, about seven seconds: four sheets drop in, one log stream per invocation, then they collapse into a single execution record. The figures in that record are illustrative. The next slide is how you put the history back together.')
NEW['observability-2'] = mk('observability-2', 'The SDK · Observability', 'Four ways to see an execution',
    xe(128, 264, 760, 600, scene('logonce', 760, 600, 8)) +
    rows([('Logging', 'A replay-aware slog logger. Records carry requestId and executionArn; step records add operationId and attempt.'),
          ('Plugins', 'Hooks on invocations, operations, attempts and child contexts. Experimental in Go.'),
          ('OpenTelemetry', 'Spans show how long each piece of work took and how the pieces relate.'),
          ('Durable Execution Telemetry', 'One JSON record per execution, exported to S3, DynamoDB or CloudWatch Logs.')], 936, 264, 856),
    'No clicks; the panel plays once, about seven seconds, and illustrates the first row. Invocation 1 logs two lines. Invocation 2 replays the handler from the top: the same two log calls run again but are suppressed, and only the new line is written, so the output has three lines, not five. ctx.Logger() and the step context logger return a standard *slog.Logger. Bring your own slog.Handler with durable.WithLogHandler, or emit replayed records tagged replay=true with durable.WithReplayLogMode(durable.ReplayLogModeEmit). Plugins are the extension point the other tools build on. Telemetry answers whole-execution questions (which step failed, how many attempts, how long) as ordinary queries; there is also an MCP server for read-only agent access. Go has no OTLP exporter.')
NEW['go-race'] = mk('go-race', 'Go · Concurrency', 'Why a context has one owner',
    xe(128, 264, 1173, 688, emb('go-concurrency')) +
    rows([('Scheduling is not deterministic', 'Two goroutines would claim operation IDs in whatever order they happen to run.'),
          ('So the SDK refuses', 'An operation called from another goroutine fails with ErrWrongGoroutine before it claims an ID.')], 1349, 264, 443),
    'No clicks; the race plays once, about twelve seconds. It shows what would happen if the SDK allowed plain goroutines on one context. First run: A reaches the claim line first and takes operation 1, B takes operation 2, both results are recorded. Replay: B wins the race, lands in operation 1 and is handed the result recorded for the charge; the slots shake and turn pink. Determinism says operations must be claimed in the same order every time; goroutine scheduling is not deterministic, so each context is owned by the goroutine that created it. A plain go statement or an errgroup.Go callback that touches the parent context fails with durable.ErrWrongGoroutine, and nothing is recorded. The check is on in every default build; the durablenocheck build tag compiles it out.')
NEW['go-concurrency'] = mk('go-concurrency', 'Go · Concurrency', 'durable.Go replaces the go keyword',
    panel(code_p('go-concurrency'), 900) +
    rows([('durable.Go returns a Future', 'It claims the child operation on the calling goroutine, then runs the body on a new one with a fresh child context.'),
          ('Await with combinators', 'All, AllSettled, Any, Race, Join, Select. Each records its outcome as one operation.'),
          ('Fan out with Map and Parallel', 'Every item or branch gets its own child context, tuned with WithMaxConcurrency and WithCompletion.')], 1076, 264, 716),
    'The fix for the race on the previous slide. The claim happens synchronously on the caller, the work happens concurrently on a child context. Inside the body use child, never the outer ctx. Do not write a sequence of Result calls that returns on the first error: the other futures are left unawaited and their progress is not checkpointed when the invocation suspends. Use a combinator; replay then returns the same outcome. A future nothing awaits still records its start (an unawaited InvokeAsync still invokes), but the invocation does not wait for it. For fan-out over data, Map with WithMaxConcurrency is usually what you want; the default completion policy is fail-fast, and CompletionConfig changes that.')
NEW['go-errors'] = mk('go-errors', 'Go · Errors', 'Errors and the suspension signal',
    xe(128, 264, 716, 688, emb('go-errors')) +
    stmt('A non-nil error from an operation is one of three things. Handle the ones you recognize, and return everything else unchanged.', 908, 400, 884),
    'No clicks; the sorter plays once, about nine seconds. A *StepError lands in typed failure: match it with errors.As. A plain error lands in the second bin: an invalid argument or option, such as a Wait under one second, so fix the call. The third token is the suspension signal: it lands, then shoots straight back up, because the only correct thing to do with it is return it. In Go, suspension travels through the error return. That is why if err != nil { return zero, err } is not just style here: it is how the invocation ends cleanly at a Wait or a callback.')
NEW['go-errors-2'] = mk('go-errors-2', 'Go · Errors', 'Match on ErrorType, not on your own type',
    panel(code_p('go-errors'), 900) +
    rows([('Rebuilt from the checkpoint', 'A typed failure carries the recorded ErrorType and Message, not your original error value.'),
          ('errors.As finds the SDK type', '*durable.StepError, or *durable.OperationError for any operation.'),
          ('Sentinels use errors.Is', 'ErrCallbackTimedOut, ErrWrongGoroutine, ErrWrongContext.')], 1076, 264, 716),
    'The pattern in code. A typed failure is rebuilt from the checkpoint, identically on the first run and on replay, so errors.As against your own CardDeclinedError will not match; compare stepErr.ErrorType. ErrorType is "Error" for errors.New and fmt.Errorf, and the Go type name otherwise. The default branch returns err untouched, which is what lets the suspension signal through. When the handler itself returns an error, the execution settles FAILED.')
NEW['go-tooling'] = mk('go-tooling', 'Go · Tooling', 'durablelint: catch it before you deploy',
    xe(128, 264, 1664, 376, emb('go-tooling')) +
    cols([('Five analyzers', 'The fifth, durablechildctx, flags a captured context used where a function has its own.'),
          ('Runs where you lint', 'As durablelint ./..., as go vet -vettool, or as a golangci-lint v2 plugin.'),
          ('Suppress one finding', 'A //durable:ignore comment on the line, or the line above it.')], 128, 692, 1664),
    'No clicks; the scan plays once, about eight seconds. Line 2, time.Now() in orchestration code: durablenondeterminism. Line 3, a go statement around a durable operation: durablegoroutine. Line 6, writing to a captured variable inside a step: durableclosure. Line 7, an operation created inside a step body: durablenestedop. The findings illustrate what each rule is for; they are not captured output from a real run. durablelint lives in its own module, github.com/aws/aws-durable-execution-sdk-go/analysis, Go 1.25 or later.')
NEW['go-test'] = mk('go-test', 'Go · Tooling', 'durabletest: a one-hour wait in a unit test',
    '<div style="width:800px;display:flex;flex-direction:column;gap:32px">\n' + panel(code_p('go-tooling'), 800) +
    '<ul style="font-size:24px;line-height:1.4;color:#3D4852">\n<li>RunUntilComplete invokes the handler as many times as the execution needs</li>\n<li>result.Operations lists every recorded operation</li>\n<li>SendCallbackSuccess resolves a callback the handler is blocked on</li>\n<li>NewCloudRunner runs the same assertions against a deployed function</li>\n</ul>\n</div>\n' +
    xe(976, 264, 816, 418, scene('testrun', 820, 420, 6.5)),
    'No clicks; the panel plays once, about five seconds. The local runner executes the handler in process: invocation 1 runs up to the one-hour Wait, the runner completes the timer itself instead of waiting, invocation 2 replays and finishes, and the test reads the result. No network and no AWS credentials. The runner checks each checkpoint against the service limits, for example a step result over 262,144 bytes. Because result.Operations lists every recorded operation, a test can assert that a step ran once across replays. The runner takes no testing handle, so the same calls work in a plain main with go run.')
def step_scene(name, bw, bh, off, end, clear=0, defs=''):
    # One click of a multi-step scene: the timeline starts at `off`, so what came before is already drawn, and holds at `end`.
    s = scene(name, bw, bh, end, clear, defs)
    a = 'frame(Math.min((n-T0)/1000,END))'
    assert a in s
    return s.replace(a, f'frame(Math.min({off}+(n-T0)/1000,END))') if off else s
def caption(cid, lead, text):
    b = f'<b><span style="color:#006E8A">{lead}</span></b> ' if lead else ''
    return f'<p id="{cid}" style="position:absolute;left:128px;top:920px;width:1664px;font-size:30px;line-height:1.3;color:#101820">{b}{text}</p>\n'

# The DynamoDB table, one click per step. The dark div under the canvas keeps the panel in place while slides change.
DATA_STEPS = [
    (0, 2.44, 'Step 1 of 4; it plays on arrival and three clicks follow. One DynamoDB table holds everything a user uploaded, in that user\'s partition: the partition key is user#, then the Cognito subject. An upload writes one item whose sort key starts with #file#; it carries the status, the labels and the thumbnail key. User 42, beach.jpg and the two labels are made up for the slide; the key layout is the one in internal/awsdrive/catalog.go.'),
    (2.8, 4.84, 'Step 2 of 4. When the upload is published, each label adds two kinds of item: an index item, #index#, then the search term, then the object key, which is what search reads, and a label item, #label#, then the label, which feeds the suggestions under the search box. A label with aliases gets one index item per alias as well, so a search for an alias is the same single query.'),
    (5.2, 7.6, 'Step 3 of 4, the important part for durable execution: the indexing step runs again, as it would on a retry, and writes the same keys, so the table still has five items. Every item is keyed by what it describes, which is what makes that step safe to retry and replay.'),
    (8, 11.3, 'Step 4 of 4, search: GET /query?index=sea is an API Gateway integration straight onto DynamoDB, a query for sort keys that begin with #index#sea.'),
]
for k, (off, end, note) in enumerate(DATA_STEPS):
    sid = 'demo-data' if k == 0 else f'demo-data-{k + 1}'
    NEW[sid] = mk(sid, 'Demo · Data', 'How PrimeDrive stores an upload',
        '<div id="data-panel" style="position:absolute;left:128px;top:264px;width:1664px;height:640px;background:#101820;border-radius:16px"></div>\n' +
        xe(128, 264, 1664, 640, step_scene('dynamo', 1664, 640, off, end)) +
        '<p id="data-note" style="position:absolute;left:128px;top:920px;width:1664px;font-size:24px;line-height:1.4;color:#3D4852">One DynamoDB table. The user ID and file name here are examples.</p>\n',
        note)
    LOGOS[sid] = LOGOS['demo-data']

# The demo architecture, drawn the way AWS reference architectures are: service icons in their own colours inside an
# AWS Cloud box, the user outside it, right-angled arrows. The picture is a still SVG; a transparent canvas over it
# draws the flow of the current step, so the diagram itself never leaves the screen between clicks.
AX, AY = 128, 264
ARCH_N = {   # key: (x, y, size, icon, service, resource)
    'api':  (400, 150, 72, 'aws-api-gateway', 'Amazon API Gateway', 'REST API'),
    's3':   (400, 490, 72, 'aws-s3', 'Amazon S3', 'uploads bucket'),
    'pre':  (700, 150, 72, 'aws-lambda', 'AWS Lambda', 'presigner'),
    'rev':  (700, 320, 72, 'aws-lambda', 'AWS Lambda', 'reviewer'),
    'pipe': (1020, 490, 96, 'aws-lambda', 'AWS Lambda', '<b><span style="color:#006E8A">durable function</span></b>'),
    'ddb':  (1420, 150, 72, 'aws-dynamodb', 'Amazon DynamoDB', 'catalog table'),
    'rek':  (1420, 320, 72, 'aws-rekognition', 'Amazon Rekognition', 'labels and moderation'),
    'thumb': (1420, 490, 72, 'aws-lambda', 'AWS Lambda', 'thumbnailer'),
}
ARCH_E = {   # arrows; each ends 16 px short of its target so the head lands on the icon's edge
    'u-api':   [(132, 320), (240, 320), (240, 150), (348, 150)],
    'u-s3':    [(132, 320), (240, 320), (240, 490), (348, 490)],
    'api-pre': [(442, 150), (648, 150)],
    'api-rev': [(442, 150), (560, 150), (560, 320), (648, 320)],
    'api-ddb': [(400, 108), (400, 62), (1420, 62), (1420, 98)],
    's3-pipe': [(442, 490), (956, 490)],
    'rev-pipe': [(742, 320), (1020, 320), (1020, 426)],
    'pipe-rek': [(1074, 490), (1220, 490), (1220, 320), (1368, 320)],
    'pipe-thumb': [(1074, 490), (1368, 490)],
    'pipe-ddb': [(1074, 490), (1220, 490), (1220, 150), (1368, 150)],
}
ARCH_STEPS = [   # lead, caption, arrows in the order they draw, icons to ring, speaker notes
    ('', 'One durable function replaces the Step Functions state machine and its trigger function.', [], [],
     'The architecture, drawn the AWS way: service icons inside the AWS Cloud box, the user outside it. It arrives without highlights; each of the next four clicks lights up one flow in teal. '
     'Not drawn: Cognito signs users in and API Gateway checks the token on every route; thumbnails go to a second bucket. '
     'Infrastructure facts if asked, from infra/lib/constructs/upload-pipeline.ts: durableConfig is the only difference from a regular function; execution timeout is the 24 hour review timeout plus one hour, retention 14 days, and one invocation is limited to one minute.'),
    ('Upload', 'The browser gets a presigned URL from the API and puts the file straight into S3.', ['u-api', 'api-pre', 'u-s3'], ['api', 'pre', 's3'],
     'Step 1 of 4, upload: the browser asks the API for a presigned URL (the presigner function) and puts the file straight into S3.'),
    ('Pipeline', 'S3 starts the durable function. It analyses, makes a thumbnail and records the result.', ['s3-pipe', 'pipe-rek', 'pipe-thumb', 'pipe-ddb'], ['s3', 'pipe', 'rek', 'thumb', 'ddb'],
     'Step 2 of 4, the pipeline: S3 invokes the durable function through its live alias, with no trigger function in between. It registers the upload, runs the Rekognition steps while a plain Lambda makes the thumbnail, and writes to DynamoDB.'),
    ('Review', 'The execution suspends. The reviewer function sends the callback and it resumes.', ['u-api', 'api-rev', 'rev-pipe'], ['api', 'rev', 'pipe'],
     'Step 3 of 4, review: when the analysis gives a reason, the execution suspends. POST /reviews reaches the reviewer function, which reads the callback ID from the file record and calls SendDurableExecutionCallbackSuccess; the same execution resumes.'),
    ('Search', 'API Gateway queries DynamoDB directly, with no function in between.', ['u-api', 'api-ddb'], ['api', 'ddb'],
     'Step 4 of 4, search: GET /query, /files and /labels are API Gateway integrations straight onto DynamoDB, with no function at all.'),
]
def arch_base():
    ink, line = '#232F3E', '#687078'
    o = (f'<svg id="arch-base" aria-label="PrimeDrive architecture on AWS" style="position:absolute;left:{AX}px;top:{AY}px" width="1664" height="640" viewBox="0 0 1664 640">'
         '<rect x="0.5" y="0.5" width="1663" height="639" rx="16" fill="#FFFFFF" stroke="#DDD8CC"/>'
         f'<rect x="200" y="16" width="1440" height="608" fill="none" stroke="{ink}" stroke-width="2"/>'
         f'<rect x="200" y="16" width="40" height="40" fill="{ink}"/>'
         '<path fill="#FFFFFF" d="M211 43 a6 6 0 0 1 0.6 -12 a8.5 8.5 0 0 1 16.4 -1.6 a6.8 6.8 0 0 1 0.8 13.6 Z"/>')
    for pts in ARCH_E.values():
        (x1, y1), (x2, y2) = pts[-2], pts[-1]
        d = abs(x2 - x1) + abs(y2 - y1); ux, uy = (x2 - x1) / d, (y2 - y1) / d
        o += (f'<path fill="none" stroke="{line}" stroke-width="3" stroke-linejoin="round" d="M' + ' L'.join(f'{x} {y}' for x, y in pts) + '"/>'
              f'<path fill="{line}" d="M{x2 + ux * 13:g} {y2 + uy * 13:g} L{x2 - ux * 3 - uy * 8:g} {y2 - uy * 3 + ux * 8:g} L{x2 - ux * 3 + uy * 8:g} {y2 - uy * 3 - ux * 8:g} Z"/>')
    o += f'<circle cx="96" cy="304" r="14" fill="{ink}"/><path fill="{ink}" d="M70 350 a26 24 0 0 1 52 0 Z"/>'
    for x, y, s, icon, _, _ in ARCH_N.values():
        src = rd(f'{LG}/{icon}.svg')
        col = re.search(r'fill="(#[0-9A-Fa-f]{6})"', src).group(1)
        # some marks sit in a group that is shifted inside the 80 px tile (S3 is); keep that shift
        tr = re.search(r'transform="translate\(([-\d.]+),\s*([-\d.]+)\)"', src)
        shift = f' transform="translate({float(tr.group(1)):.1f} {float(tr.group(2)):.1f})"' if tr else ''
        o += (f'<g transform="translate({x - s / 2:g} {y - s / 2:g}) scale({s / 80:g})"><rect width="80" height="80" fill="{col}"/>'
              f'<path fill="#FFFFFF" fill-rule="evenodd"{shift} d="{aws_path(icon)}"/></g>')
    o += '</svg>\n'
    lab = "position:absolute;width:320px;text-align:center;font-size:24px;line-height:1.25;color:#101820"
    o += f'<p id="arch-l-cloud" style="position:absolute;left:{AX + 252}px;top:{AY + 21}px;width:240px;font-size:24px;font-weight:600;line-height:1.25;color:#232F3E">AWS Cloud</p>\n'
    o += f'<p id="arch-l-user" style="{lab};left:{AX + 96 - 160}px;top:{AY + 358}px"><b>User</b></p>\n'
    for k, (x, y, s, _, svc, res) in ARCH_N.items():
        o += f'<p id="arch-l-{k}" style="{lab};left:{AX + x - 160}px;top:{AY + y + s // 2 + 8}px"><b>{svc}</b><br><span style="color:#3D4852">{res}</span></p>\n'
    return o
ARCH_BASE = arch_base()
for k, (lead, cap, edges, hot, note) in enumerate(ARCH_STEPS):
    sid = 'demo' if k == 0 else f'demo-{k + 1}'
    body = ARCH_BASE
    if edges:
        defs = ('var EDG=' + json.dumps([ARCH_E[e] for e in edges], separators=(',', ':')) +
                ',HOT=' + json.dumps([list(ARCH_N[h][:3]) for h in hot], separators=(',', ':')) + ';\n')
        body += xe(AX, AY, 1664, 640, scene('archflow', 1664, 640, 1.5 + len(edges) * .6, 1, defs))
    NEW[sid] = mk(sid, 'Demo · Architecture', 'PrimeDrive, the Go Durable way', body + caption(f'arch-cap-{k}', lead, cap), note)
NEW['demo-handler'] = code_slide('demo-handler', 'One handler, top to bottom', '''
func (p *Pipeline) Handle(ctx durable.Context, event events.S3Event,
) (drive.File, error) {
    upload, err := uploadFrom(event) // plain code: no step
    // ...
    file, err := p.register(ctx, upload) // two steps
    // ...
    settled, err := p.process(ctx, file) // analyze, review, publish

    // Not every error is a failure: a suspension is passed on.
    var failure *«durable.OperationError»
    if errors.As(err, &failure) {
        err = p.fail(ctx, file, err)
    }
    // ...
    return settled, err
}''',
    [('Plain code needs no step', 'Reading the S3 event is deterministic: a replay gives the same answer.'),
     ('Suspension is an error', 'When the function waits, the blocked operation returns a signal. Pass it on unchanged.'),
     ('A failure has a type', 'SDK failures match *durable.OperationError. Only then is the upload marked FAILED.')],
    'From internal/pipeline/pipeline.go. The lines marked with dots are left out to fit: the early returns on err and, at the end, the check for a run that a newer upload of the same file has overtaken; such a run changes nothing and ends as SUPERSEDED. register is two steps: one makes a random name for this run, which is nondeterministic and therefore belongs inside a step, and one writes the file record under that name. This is the whole workflow: the function is replayed from the top every time the execution resumes, and operations that already completed return their recorded result. The part that is specific to Go is the end: suspension travels through the error return, so the handler must not treat every error as a failure. It records FAILED only for errors that match durable.OperationError and returns everything else unchanged.')
NEW['demo-parallel'] = code_slide('demo-parallel', 'Two things at once, safely', '''
analysis := «durable.Go»(ctx, "analyze-image",
    func(child durable.Context) (drive.Analysis, error) {
        return p.analyze(child, upload)
    })

thumb := «durable.InvokeAsync»[thumbnail.Result](ctx, "generate-thumbnail",
    p.cfg.ThumbnailerFunction, thumbnail.Request{Source: upload.Object})

err := «durable.Join»(ctx, "await-analysis-and-thumbnail",
    []durable.Awaitable{analysis, thumb})
// ...
found, err := analysis.Result(ctx)
made, err := thumb.Result(ctx)''',
    [('durable.Go replaces go', 'The body gets its own goroutine and a child context. Use child, never ctx.'),
     ('InvokeAsync calls a Lambda', 'The call is checkpointed, so a replay never invokes the thumbnailer twice.'),
     ('Join awaits both', 'Awaiting them together lets each branch reach a checkpoint before a suspension.')],
    'From internal/pipeline/analysis.go; this was the Parallel state of the state machine. durable.Go is the replay-safe go statement from the concurrency slides. InvokeAsync is generic over the result type of the other function. Join awaits futures of different result types; after it both Result calls return at once. Inside analyze, detect-labels and detect-moderation are two steps with a retry strategy: four attempts, starting at two seconds, doubling, and no retry for an unsupported image.')
NEW['demo-review'] = code_slide('demo-review', 'A day-long wait is one call', '''
decision, err := «durable.WaitForCallback»[drive.Decision](
    ctx, "human-review",
    func(sc durable.StepContext, callbackID string) error {
        return p.catalog.SaveFile(sc,
            file.AwaitingReview(callbackID, reasons))
    },
    durable.WithCallbackTimeout(p.cfg.ReviewTimeout),
    durable.WithCallbackSerdes(durable.JSONSerdes),
    durable.WithSubmitterRetry(retryTransient()),
)

var timedOut *«durable.CallbackTimeoutError»
if errors.As(err, &timedOut) {
    return drive.Decision{Approved: false,
        Note: "no reviewer responded in time"}, nil
}
return decision, err''',
    [('The submitter runs once', 'It saves the callback ID on the file record. Then the execution suspends.'),
     ('Nothing runs while waiting', 'POST /reviews sends the decision to that callback ID and the execution resumes here.'),
     ('A timeout is a decision', 'errors.As matches CallbackTimeoutError and the upload is rejected.')],
    'From internal/pipeline/review.go. Review only happens when the policy gives a reason: a moderation flag, or no label at 80 percent confidence; otherwise the upload is approved without waiting. POST /reviews names a file, never a callback ID: the reviewer function reads the ID from the caller\'s own file record and calls SendDurableExecutionCallbackSuccess with the decision as JSON. The submitter is a step with a retry strategy; it does not retry when a newer upload of the same file has taken the record over. The review timeout is 24 hours by default. After this, an approved upload is indexed with durable.Map, one step per label, and a rejected one is marked and then removed with durable.Parallel.')

# id -> (group, index, flags)
ORDER = [('cover', None, 0, ''), ('why', None, 0, ''),
    ('first', 'how', 0, ''), ('replay', 'how', 1, ''), ('replay-live', 'how', 1, ''), ('determinism', 'how', 2, ''),
    ('sdk-ops', 'sdk', 0, 'i'), ('operations', 'sdk', 0, ''), ('waiting', 'sdk', 0, ''), ('waiting-2', 'sdk', 0, ''), ('waiting-3', 'sdk', 0, ''),
    ('sdk-errors', 'sdk', 1, 'i'), ('errors', 'sdk', 1, ''), ('errors-2', 'sdk', 1, ''), ('errors-3', 'sdk', 1, ''),
    ('sdk-state', 'sdk', 2, 'i'), ('state', 'sdk', 2, ''), ('state-2', 'sdk', 2, ''),
    ('sdk-obs', 'sdk', 3, 'i'), ('observability', 'sdk', 3, ''), ('observability-2', 'sdk', 3, ''),
    ('go', 'go', -1, 't'),
    ('go-i-shape', 'go', 0, 'i'), ('go-shape', 'go', 0, ''),
    ('go-i-conc', 'go', 1, 'i'), ('go-race', 'go', 1, ''), ('go-concurrency', 'go', 1, ''),
    ('go-i-errors', 'go', 2, 'i'), ('go-errors', 'go', 2, ''), ('go-errors-2', 'go', 2, ''),
    ('go-i-tools', 'go', 3, 'i'), ('go-tooling', 'go', 3, ''), ('go-test', 'go', 3, ''),
    ('go-i-diff', 'go', 4, 'i'), ('go-gotchas', 'go', 4, ''),
    ('demo', 'demo', 0, ''), ('demo-2', 'demo', 0, ''), ('demo-3', 'demo', 0, ''), ('demo-4', 'demo', 0, ''), ('demo-5', 'demo', 0, ''),      # the architecture, one flow per click
    ('demo-handler', 'demo', 1, ''), ('demo-parallel', 'demo', 1, ''), ('demo-review', 'demo', 1, ''),
    ('demo-data', 'demo', 2, ''), ('demo-data-2', 'demo', 2, ''), ('demo-data-3', 'demo', 2, ''), ('demo-data-4', 'demo', 2, ''),      # the table, one step per click
    ('demo-video', None, 0, ''),      # hand-written placeholder: the recorded demo plays here
    ('wrap', None, 0, '')]      # flag h would keep a slide in the deck but skip it when presenting

changed = []
for n, (sid, group, idx, fl) in enumerate(ORDER):
    if not group: continue
    text = NEW[sid] if sid in NEW else bk(sid)
    if group == 'go' and 'i' not in fl:
        if sid != 'go':
            text = text.replace('<div style="display:flex;flex-direction:column;gap:12px">', '<div style="display:flex;flex-direction:column;gap:12px;padding:0px 0px 0px 48px">', 1)
    nxt = ORDER[n + 1] if n + 1 < len(ORDER) else None
    if nxt and nxt[1] == group and group != 'go':
        trans = 'magic'
    elif group == 'go' and 'i' in fl:
        # During a fade the deck draws the leaving slide without its embeds, which would uncover the start-state
        # arc under the canvas. With "magic" the arc SVG is matched by id and the next slide's arc shows at once.
        trans = 'magic'
    else:
        trans = 'fade'
    prv = ORDER[n - 1] if n else None
    pidx = None
    if group == 'go' and prv and prv[1] == 'go' and prv[2] != idx: pidx = prv[2]
    if sid in LOGOS: text = text.replace('<aside>', logo_svg(*LOGOS[sid]) + '<aside>', 1)
    if 'h' in fl: text = text.replace(f'<section id="{sid}"', f'<section id="{sid}" hidden', 1)
    wr(sid, finish(text, group, idx, trans, dark=('d' in fl), intro=('i' in fl), teal=('t' in fl), prev=pidx))
    changed.append(sid)

idx_path = S + '/deck/project/deck.json'
d = json.load(open(idx_path, encoding='utf-8'))
d['order'] = [o[0] for o in ORDER]
d['sections']['s3']['start'] = 'sdk-ops'
d['sections']['s5']['start'] = 'demo'
json.dump(d, open(idx_path, 'w', encoding='utf-8'), indent=2, ensure_ascii=False)
print(len(ORDER), 'slides in order;', len(changed), 'files written')
print(json.dumps({f'project/slides/{c}.html': f'project/slides/{c}.html' for c in changed}))
