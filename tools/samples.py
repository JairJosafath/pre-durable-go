#!/usr/bin/env python3
"""Downloads sample photos for a PrimeDrive demo into a folder.

    tools/samples.py <folder>

About twenty photos are picked at random from Wikimedia Commons, one per
topic, so every download gives a different set. A few more always show beer
or wine: those are the ones the pipeline should send to a reviewer.

Every photo is freely licensed. SOURCES.txt in the folder lists where each
one came from, which is where its author and licence are.
"""

import json
import random
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

API = "https://commons.wikimedia.org/w/api.php"
# Wikimedia asks every client to say what it is.
AGENT = "primedrive-samples/1.0 (demo seeding script)"
WIDTH = 960  # one of the thumbnail widths Commons serves
PAUSE = 1  # seconds between photos, to be a polite client
WAITS = [5, 15, 30, None]  # seconds to wait after each refused attempt; then give up
TOO_MANY_REQUESTS = 429

# One random photo per topic, named <topic>.jpg.
TOPICS = [
    "cat", "dog", "horse", "bird", "butterfly", "flower", "tree", "mountain",
    "beach", "lake", "bridge", "castle", "bicycle", "car", "train", "airplane",
    "boat", "pizza", "coffee", "guitar",
]  # fmt: skip

# Photos that should need a review, named <name>.jpg. These are fixed, not
# random: each one clearly shows alcohol.
REVIEW = {
    "beer": "File:Chilled glass of beer.jpg",
    "beer-bottles": "File:Grolsch Beer Bottles.jpg",
    "red-wine": "File:Bottle and glass of red wine.jpg",
    "wine-bottles": "File:Dr Loosen and Robert Weil wine bottles.jpg",
}


def get(url, params=None):
    if params:
        url += "?" + urllib.parse.urlencode(params)

    request = urllib.request.Request(url, headers={"User-Agent": AGENT})

    for wait in WAITS:
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                return response.read()
        except urllib.error.HTTPError as problem:
            # Commons limits how fast a client may download. Wait and retry.
            if problem.code != TOO_MANY_REQUESTS or wait is None:
                raise

            time.sleep(float(problem.headers.get("Retry-After") or wait))

    raise AssertionError("unreachable: the last wait is None")


def pages(**query):
    """Asks Commons for files, with a download URL for each."""
    answer = json.loads(get(API, {
        "action": "query", "format": "json", "prop": "imageinfo",
        "iiprop": "url|mime", "iiurlwidth": WIDTH, **query,
    }))  # fmt: skip

    found = answer.get("query", {}).get("pages", {}).values()

    return [page for page in found if page.get("imageinfo")]


def random_photo(topic):
    """Picks one of the photos Commons editors rated as good for a topic."""
    found = pages(
        generator="search", gsrnamespace=6, gsrlimit=40,
        gsrsearch=f'filemime:image/jpeg incategory:"Quality images" {topic}',
    )  # fmt: skip

    return random.choice(found) if found else None


def named_photo(title):
    found = pages(titles=title)

    return found[0] if found else None


def save(folder, name, page):
    info = page["imageinfo"][0]
    (folder / f"{name}.jpg").write_bytes(get(info.get("thumburl") or info["url"]))

    return f"{name}.jpg  {info['descriptionurl']}"


def main():
    if len(sys.argv) != 2:
        sys.exit(__doc__)

    folder = Path(sys.argv[1])
    folder.mkdir(parents=True, exist_ok=True)

    wanted = [(topic, random_photo, topic) for topic in TOPICS]
    wanted += [(name, named_photo, title) for name, title in REVIEW.items()]

    sources = []
    for name, find, what in wanted:
        try:
            page = find(what)
            if page is None:
                print(f"  skipped {name}: nothing found")
                continue

            sources.append(save(folder, name, page))
            print(f"  {name}.jpg")
        except OSError as problem:  # one photo that fails does not stop the rest
            print(f"  skipped {name}: {problem}")

        time.sleep(PAUSE)

    if not sources:
        sys.exit("No photo could be downloaded. Is commons.wikimedia.org reachable?")

    (folder / "SOURCES.txt").write_text(
        "Sample photos from Wikimedia Commons. Each page names the author and the licence.\n\n"
        + "\n".join(sources) + "\n"
    )  # fmt: skip
    print(f"{len(sources)} photos in {folder}")


if __name__ == "__main__":
    main()
