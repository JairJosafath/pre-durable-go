#!/usr/bin/env python3
"""Builds the offline copy of the talk: one HTML file that plays the deck
in any browser, with its animations, and needs no network.

    python3 talk/offline/build.py

It reads the slides in talk/deck/project (what talk/gen.py writes and the
published deck holds) and writes durable-functions-in-go.html next to this
script, with player.css, player.js, the fonts, the icons and the pictures
all inside it.

Three things differ from the published deck:

  - The recorded demo is three screenshots from it, each on a slide of its
    own, so the file stays small and plays everywhere.
  - The icons of the topic bar are Lucide's (ISC licence). The published
    deck draws them with the icon font of the Slides runtime, which is not
    ours to copy.
  - The hidden slide is left out, as it is when the deck is presented.
"""

import base64
import json
import re
from pathlib import Path

HERE = Path(__file__).resolve().parent
PROJECT = HERE.parent / "deck" / "project"
ASSETS = HERE / "assets"
OUTPUT = HERE / "durable-functions-in-go.html"

# The typefaces of the deck, as variable fonts (latin subset), from Google
# Fonts. Both are under the SIL Open Font License.
FONTS = [
    ("IBM Plex Sans", "IBMPlexSans.woff2", "100 700"),
    ("JetBrains Mono", "JetBrainsMono.woff2", "100 800"),
]

# The name a slide gives an icon, and the Lucide icon that stands in for it.
ICONS = {
    "Activity": "activity",
    "CheckCircle": "circle-check",
    "Cloud": "cloud",
    "Code": "code",
    "Database": "database",
    "Play": "play",
    "Settings": "settings",
    "Warning": "triangle-alert",
}
ICON_STROKE = 1.75

# The slide that plays the recording, and what replaces it here.
RECORDING = "demo-play"
SCREENSHOTS = [
    (
        "demo-1.webp",
        "An upload nobody has to look at: one invocation, every operation checkpointed, ready in 2.5 seconds.",
        "Screenshot 1 of 3 from the recorded demo, against the deployed stack. airplane.jpg was uploaded and the dev "
        "panel on the right shows its durable execution: every operation by name with its duration, one invocation, "
        "no replays.",
    ),
    (
        "demo-2.webp",
        "Rekognition flags beer.jpg. The execution suspends on a callback until a reviewer answers.",
        "Screenshot 2 of 3. make seed is uploading the sample photos. beer.jpg was flagged as an alcoholic beverage, "
        "so it waits under \"Waiting for your review\". In the dev panel its execution stopped at human-review: the "
        "callback is waiting, and no function is running. castle.jpg is still being analysed.",
    ),
    (
        "demo-3.webp",
        "After the approval: a second invocation replays the finished steps, then indexes the labels.",
        "Screenshot 3 of 3. red-wine.jpg was approved after 33 seconds. Its execution shows two invocations and one "
        "replay: the handler ran again from the top, the steps before the callback returned their stored results, "
        "and index-labels ran for the first time.",
    ),
]

# Speaker notes that talk about the recording, reworded for the screenshots.
NOTES = {
    "demo-video": "Title card for the demo. The audience has seen the architecture, the code and the data. The next "
    "three slides are screenshots from the recorded PrimeDrive demo: an upload that needs no review, one that waits "
    "for a reviewer, and the same kind of upload after it was approved.",
}

SECTION = re.compile(r"<section\b([^>]*)>(.*)</section>", re.S)
EMBED = re.compile(r"<x-embed\b([^>]*)>(.*?)</x-embed>", re.S)
ASIDE = re.compile(r"<aside>(.*?)</aside>", re.S)


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def data_uri(path: Path, media_type: str) -> str:
    return f"data:{media_type};base64,{base64.b64encode(path.read_bytes()).decode('ascii')}"


def plain(text: str) -> str:
    """The text of an <aside>, without markup."""
    text = re.sub(r"<[^>]+>", "", text)
    for entity, char in (("&lt;", "<"), ("&gt;", ">"), ("&quot;", '"'), ("&#39;", "'"), ("&amp;", "&")):
        text = text.replace(entity, char)

    return " ".join(text.split())


def screenshot_slide(number: int, picture: str, caption: str, notes: str) -> dict:
    """A slide that shows one screenshot of the demo, with a line under it."""
    html = (
        f'<section id="demo-shot-{number}" data-transition="fade" style="background:#101820">'
        f'<img src="{data_uri(ASSETS / "shots" / picture, "image/webp")}" alt="{caption}" '
        'style="position:absolute;left:216px;top:32px;width:1488px;height:930px;border-radius:12px">'
        '<p style="position:absolute;left:128px;top:986px;width:1664px;text-align:center;'
        f'font-size:28px;line-height:1.4;color:#A9B7C4">{caption}</p>'
        "</section>"
    )

    return {"id": f"demo-shot-{number}", "transition": "fade", "html": html, "notes": notes}


def slide(slide_id: str, embeds: list) -> dict | None:
    """One slide of the deck, or None for a slide that is hidden."""
    source = read(PROJECT / "slides" / f"{slide_id}.html")
    attributes, body = SECTION.search(source).groups()

    if re.search(r"\shidden(\s|=|$)", attributes):
        return None

    transition = re.search(r'data-transition="(\w+)"', attributes)
    aside = ASIDE.search(body)
    body = ASIDE.sub("", body)

    # A scene is a small page of its own. It is kept apart from the slide,
    # so its script only ever runs inside the frame the player gives it.
    def keep_apart(match: re.Match) -> str:
        embeds.append(match.group(2).strip())

        return f'<x-embed{match.group(1)} data-embed="{len(embeds) - 1}"></x-embed>'

    body = EMBED.sub(keep_apart, body)

    return {
        "id": slide_id,
        "transition": transition.group(1) if transition else "",
        "html": f"<section{attributes}>{body.strip()}</section>",
        "notes": NOTES.get(slide_id) or (plain(aside.group(1)) if aside else ""),
    }


def icon(name: str) -> str:
    """A Lucide icon as inline SVG that takes its colour from the text."""
    drawing = re.search(r"<svg[^>]*>(.*)</svg>", read(ASSETS / "icons" / f"{name}.svg"), re.S).group(1)
    drawing = " ".join(drawing.split())

    return (
        '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" '
        f'stroke-width="{ICON_STROKE}" stroke-linecap="round" stroke-linejoin="round">{drawing}</svg>'
    )


def font_faces() -> str:
    return "\n".join(
        f"@font-face {{ font-family: '{family}'; font-style: normal; font-weight: {weights}; "
        f"src: url({data_uri(ASSETS / 'fonts' / file, 'font/woff2')}) format('woff2'); }}"
        for family, file, weights in FONTS
    )


def build() -> None:
    index = json.loads(read(PROJECT / "deck.json"))
    slides, embeds = [], []

    for slide_id in index["order"]:
        if slide_id == RECORDING:
            slides += [screenshot_slide(n, *shot) for n, shot in enumerate(SCREENSHOTS, start=1)]
            continue

        one = slide(slide_id, embeds)
        if one:
            slides.append(one)

    deck = {
        "title": index["title"],
        "slides": slides,
        "embeds": embeds,
        "icons": {name: icon(file) for name, file in ICONS.items()},
    }
    # Inside a <script>, "</" and "<!--" would end or confuse the element.
    payload = json.dumps(deck, ensure_ascii=False, separators=(",", ":"))
    payload = payload.replace("</", "\\u003c/").replace("<!--", "\\u003c!--")

    page = f"""<!doctype html>
<!--
  {index["title"]}: the talk as one file. Open it in a browser.
  Generated by talk/offline/build.py; change the slides, not this file.

  Icons: Lucide, ISC licence, https://lucide.dev
  Fonts: IBM Plex Sans (IBM) and JetBrains Mono (JetBrains), SIL Open Font License 1.1
-->
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{index["title"]}</title>
<style>
{font_faces()}
{read(HERE / "player.css")}</style>
</head>
<body>
<div id="stage"></div>
<div id="notes" hidden></div>
<div id="hint">← → or click to move · F full screen · N speaker notes</div>
<script id="deck" type="application/json">{payload}</script>
<script>
{read(HERE / "player.js")}</script>
</body>
</html>
"""
    OUTPUT.write_text(page, encoding="utf-8")
    print(f"{OUTPUT.relative_to(HERE.parent.parent)}: {len(slides)} slides, {len(embeds)} scenes, {len(page) // 1024} KB")


if __name__ == "__main__":
    build()
