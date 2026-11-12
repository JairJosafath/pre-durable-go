#!/bin/bash
# usage: shot.sh <slide-id> <width> <height> <t1> [t2 ...]
# Extracts the first x-embed of a slide into a standalone page and screenshots it at fixed times.
S="$(cd "$(dirname "$0")/.." && pwd)"
R="${SHOT_OUT:-/tmp/talk-render}"; mkdir -p "$R"
ID=$1; W=$2; H=$3; shift 3
python3 - "$S/deck/project/slides/$ID.html" "$R/$ID.embed.html" <<'EOF'
import re,sys
src=open(sys.argv[1],encoding='utf-8').read()
m=re.search(r'<x-embed[^>]*>(.*?)</x-embed>',src,re.S)
body=m.group(1)
open(sys.argv[2],'w',encoding='utf-8').write(body)
print('embed bytes',len(body.encode('utf-8')),'has < in script:', '<' in re.search(r'<script>(.*?)</script>',body,re.S).group(1), 'has &:', '&' in re.search(r'<script>(.*?)</script>',body,re.S).group(1))
EOF
CH="/mnt/c/Program Files/Google/Chrome/Application/chrome.exe"
WIN=$(wslpath -w "$R/$ID.embed.html" | sed 's#\\#/#g')
OUT=$(wslpath -w "$R")
PROF='C:\Users\jerem\AppData\Local\Temp\claude-render-profile'
for T in "$@"; do
  "$CH" --headless=new --disable-gpu --hide-scrollbars --user-data-dir="$PROF" --window-size=$W,$H --virtual-time-budget=400 --screenshot="$OUT\\$ID-$T.png" "file:$WIN#t=$T" >/dev/null 2>&1
done
ls -la $R/$ID-*.png | awk '{print $5, $9}'
