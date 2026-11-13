# The talk

The slides of "Durable Functions in Go" (AWS User Group NL, pre:Invent 2026).
Nothing in the application uses this folder.

| Path | What it is |
| --- | --- |
| `offline/durable-functions-in-go.html` | The whole talk as one file. Open it in a browser: it plays without a network, animations included. Arrow keys or a click move through it, `F` is full screen, `N` shows the speaker notes |
| `offline/build.py` | Builds that file from the slides: `python3 talk/offline/build.py` |
| `deck/project/` | The slides as the published deck holds them, one file per slide |
| `gen.py` | Writes most of `deck/project/` from `base/`, `scenes/` and `logos/` |
| `scenes/` | The canvas animations |
| `tools/shot.sh` | Renders one frame of a slide's animation to a picture |

The offline copy differs from the published deck in three ways: the recorded
demo is three screenshots, the topic icons are Lucide's, and the hidden
slide is left out. [offline/build.py](offline/build.py) says why.
