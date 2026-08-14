---
name: redacting-flatex-pdfs
description: Use when turning real flatexDEGIRO broker PDFs (Kauf/Verkauf trade confirmations, Fondsertragsausschüttung / Ertragsmitteilung dividend statements) into PII-free test fixtures — replacing customer name, address, Depot/Konto/Transaktion/Auftragsnummer with synthetic values while keeping each document byte-for-byte visually identical to the original.
---

# Redacting flatex PDFs into Test Fixtures

## Overview

Real flatex statements contain a customer's name, address, and account numbers. To use them as test fixtures, replace that PII with **synthetic** values while keeping the page visually identical.

**Core technique:** redact the exact PII text rectangles, re-insert synthetic text at the same position in a *base-14 font*, then **reflow the content streams** (see below — skipping this produces a fixture that looks perfect and parses wrong). The original flatex PDFs use embedded Identity-H fonts (`HerosBFO`, `CursorBFO`) throughout — including PII fields. Base-14 substitutes (Helvetica, Courier) are visually similar but technically distinct; the PII fields will render in a slightly different font than the surrounding document. For parsing purposes this does not matter — positions, sizes, and structure are preserved. Do **not** try to reuse the embedded fonts — they are Identity-H subsets and silently produce wrong glyphs for any character not already in the subset.

Tools: **PyMuPDF (`fitz`)** for redaction, **Presidio** to *identify* PII candidates, and `reflow.py` (in this directory) for the mandatory final pass.

## The reflow step is not optional

`page.insert_text()` and `add_redact_annot(text=...)` both **append** the replacement as a *new content stream* at the end of the page's `/Contents`. The page renders correctly, so a visual diff passes — but extractors that read in content-stream order rather than sorting geometrically see every replaced value drop out of its slot and reappear as a block of loose text at the bottom of the page.

`flatex-pdf-cli` uses `gxpdf`, which reads in stream order. A fixture redacted without reflowing yields text like

```
Nr. /1    Kauf    BITCOIN          <- order number gone from its slot
...
440000111                          <- ...re-emerges 30 lines later
```

which silently breaks every parser anchored on an identifier (`Nr.<order>/N`, the 9-digit `Auftrags-Nr`), and makes `Depotinhaber:` capture the *following* line. All seven fixtures in this repo were once broken exactly this way.

After redacting, always run:

```python
from reflow import reflow          # .claude/skills/redacting-flatex-pdfs/reflow.py

doc = fitz.open(src)
... redact and insert ...
reflow(doc)                        # move appended text into reading order
doc.save(out, garbage=4, deflate=True)
```

`reflow` moves each appended block to its reading-order position inside the main stream, carrying its white cover rectangle with it, and raises `ReflowError` rather than writing a half-edited stream. Rendering is unchanged (measured: <50 differing subpixels at 300 DPI, i.e. antialiasing on re-anchored glyphs).

## What to replace vs. keep

| Field (German label on the doc) | Action |
|---|---|
| Name in address block + `Depotinhaber:` (`Last, First`) | replace — synthetic person |
| Street + house no., postal code + city, `Stiege`/`Tür` | replace — synthetic address |
| `Ihre Depotnummer:` (11 digits) | replace — keep length |
| `Konto Nr.:` (11 digits) | replace — keep length |
| `Transaktion-Nr.:` (10 digits) | replace — keep length |
| `Auftragsnummer` / `Nr. …/N` (trade docs) | replace — keep `/N` suffix |
| Salutation `Herrn`/`Frau` | match the synthetic person's gender |
| flatex corporate boilerplate (company address, board/management names, FN/HRB/UID, phone, email) | **keep** — public, identical on every doc |
| Barcodes + their readable digit codes (top/left margins) | **keep** — they encode a doc-tracking ID (not name/account), and the bars can't be regenerated without breaking the visual. Their readable text is rotated and re-inserts poorly. |

Presidio (English NLP; no German spaCy model is installed) is **noisy** on German — it flags the public board members and many false positives. Use it to surface candidates, then curate by hand using the table above.

## Synthetic persona pool (PII-free)

Assign a **different** persona per fixture so the corpus exercises titles, umlauts, hyphens, gender, and varied lengths/number patterns. All fictitious:

| Fixture | Salutation / name | Address | Depot / Konto | Tx / Auftrag |
|---|---|---|---|---|
| trade 1 | Herrn / Dr. Max Mustermann | Musterstrasse 12, Stiege 1 Tür 2, 1010 Wien | 11000000011 / 11000000012 | 7000000011 / 700000011 |
| trade 2 | Frau / Erika Beispiel | Beispielweg 5, Stiege 4 Tür 11, 1020 Wien | 22000000021 / 22000000022 | 7000000022 / 800000022 |
| dividend 1 | Herrn / Johann Österreicher | Lindengasse 8, Stiege 2 Tür 5, 1070 Wien | 33000000031 / 33000000032 | 7000000033 / — |
| dividend 2 | Frau / Anna-Maria Gruber | Ahornstrasse 23, Stiege 7 Tür 3, 1150 Wien | 44000000041 / 44000000042 | 7000000044 / — |
| sparplan 1 | Herrn / Dr. Klaus Bergmann | Bergmannsgasse 17, Stiege 5 Tür 3, 1050 Wien | 55000000051 / — | — / 0005500055 |
| krypto 1 | Herrn / Dr. Stefan Berger | Kirschenallee 9, Stiege 6 Tür 1, 1090 Wien | 66000000061 / 66000000062 (Verwahrkonto 66000000063) | 6600000066 / 660000111 |
| order 1 | Herrn / Dr. Lukas Hofer | Hofergasse 4, Stiege 8 Tür 2, 1080 Wien | 77000000071 / — | — / 770000111, 770000222 |
| trade 3 | Herrn / Mag. Felix Steiner | Wagramer Str. 118, Stiege 4 Tür 311, 1220 Wien | 88000000081 / 88000000082 | 8800000088 / 880000088 |
| verkauf 1 | Frau / Mag. Sophie Wallner | Wallnergasse 31, Stiege 3 Tür 9, 1030 Wien | 99000000091 / 99000000092 | 9900000099 / 990000099 |
| trade 3 | Herrn / Mag. Felix Steiner | Wagramer Str. 118, Stiege 4 Tür 311, 1220 Wien | 88000000081 / 88000000082 | 8800000088 / 880000088 |

Every fixture has its own name, address and number block, so a cross-fixture mix-up cannot pass a test unnoticed. Keep it that way when adding one.

Keep digit-string **lengths equal** to the originals so mono-column alignment is preserved. Umlauts (ä ö ü Ö) and hyphens are fine in both Helvetica and Courier (WinAnsi).

## Reference implementation

`page.search_for(old)` gives exact rects; the covering span gives font + baseline. Redact all rects, `apply_redactions()`, then re-insert.

```python
import fitz
from reflow import reflow
# Older statements name their fonts with the literal base-14 names instead of
# using the embedded subsets. Match longest key first so "Courier-Bold" wins
# over "Courier", and raise on an unknown font rather than defaulting: silently
# falling back to "helv" renders a mono column in a proportional face and
# destroys its alignment.
FONTMAP = {"CursorBFO-Regular":"cour","CursorBFO-Bold":"cobo",
           "HerosBFO-Regular":"helv","HerosBFO-Bold":"hebo","OfficinaSans":"helv",
           "Courier-Bold":"cobo","Courier":"cour",
           "Helvetica-Bold":"hebo","Helvetica":"helv"}
def b14(f):
    for k in sorted(FONTMAP, key=len, reverse=True):
        if k in f: return FONTMAP[k]
    raise SystemExit(f"unmapped font {f!r} — add it to FONTMAP")

def redact(src, out, replacements):           # replacements: {old_text: synthetic}
    doc = fitz.open(src)
    for page in doc:
        spans = [(fitz.Rect(s["bbox"]), s["origin"], s["font"], s["size"], s["text"])
                 for b in page.get_text("dict")["blocks"]
                 for l in b.get("lines", []) for s in l["spans"]]
        def span_at(r):
            c = fitz.Point((r.x0+r.x1)/2, (r.y0+r.y1)/2)
            return next((s for s in spans if s[0].contains(c)), None)
        ins, claimed = [], []
        for old in sorted(replacements, key=len, reverse=True):   # longest first
            new = replacements[old]
            for r in page.search_for(old):
                s = span_at(r)
                if not s: continue
                # a value can occur inside the margin barcode's readable digits
                if s[4].strip().isdigit() and len(s[4].strip()) > len(old): continue
                # same text already claimed by a longer key (name vs name+comma)
                if any(abs(c.y0-r.y0) < 1.0 and c.x0-1 <= r.x0 <= c.x1 for c in claimed):
                    continue
                claimed.append(fitz.Rect(r))
                pad = r.height * 0.18                    # see "collateral damage"
                box = fitz.Rect(r.x0+0.3, r.y0+pad, r.x1-0.3, r.y1-pad)
                page.add_redact_annot(box, fill=(1,1,1))       # erase original
                ins.append((r.x0, s[1][1], new, b14(s[2]), s[3]))  # x, baseline_y, text, font, size
        page.apply_redactions()
        for x, by, new, fn, sz in ins:
            page.insert_text((x, by), new, fontname=fn, fontsize=sz, color=(0,0,0))
    reflow(doc)                                # mandatory; see above
    doc.save(out, garbage=4, deflate=True)
```

## Verify before claiming done

1. **Residual scan** — confirm no original token survives in the text:
   `"".join(p.get_text() for p in fitz.open(out))` must not contain any original name fragment or number.
2. **Visual diff** — render before/after to PNG (`page.get_pixmap(dpi=200)`) and confirm layout + fonts match. Check the address block (Helvetica path) and a mono body line (Courier path) at high DPI.
3. **Content check** — build the expected text by applying the replacement map to the *original's* `get_text(sort=True)`, and compare it against the redacted file's. These must hold the same tokens. Anything missing means `apply_redactions` ate a neighbour; a token in a different position is usually a harmless sort flip between two columns, so compare the multiset (`collections.Counter`) before worrying.

   Do **not** test "unsorted text == sorted text" as a proxy for correct ordering. flatex draws the margin barcodes out of reading order, so the pristine originals fail that check too — it flags nothing useful.

4. **Diff the parsed JSON** — the strongest check, and the one that catches everything above at once. Run the CLI on the original *and* the redacted copy and diff:

   ```bash
   flatex-pdf-cli -quiet -include-metadata original.pdf   > /tmp/a.json
   flatex-pdf-cli -quiet -include-metadata redacted.pdf   > /tmp/b.json
   diff <(jq -S . /tmp/a.json) <(jq -S . /tmp/b.json)
   ```

   The transaction count must be equal and **the only differing leaves may be the fields you deliberately replaced**. A field that goes *empty* is a bug: that is how the clipped salutation comma was found — `depot_holder` silently became `""` because the extractor's fallback needs the comma the redaction had eaten.

5. **Run the suite** — `go test ./internal/parser/ -run TestAllFixturesParse`.

## Common mistakes

- **Skipping the reflow pass** → the page renders perfectly and a pixel diff passes, but every replaced identifier is out of position in stream order, so `gxpdf` hands the parser blanks. Trusting a visual diff alone is what let this ship.
- **Stranding an unreplaced neighbour inside a replaced block** → `reflow` only
  re-anchors the lines you actually replaced. Replace five of the six address
  lines and the sixth stays behind at its old stream position, where the next
  text in stream order absorbs it. On the Verkauf document that left
  `ÖSTERREICH` glued to the right column's `Ausf.platz/-art    Tradegate`, so
  `execution_venue` parsed as `TradegateSTERREICH` while the page still
  rendered perfectly. Fix: map the leftover line to itself so it travels with
  its block. Only the JSON diff (check 4) catches this — the residual scan, the
  token multiset and the pixel diff all pass.
- **A self-mapped or short key matching elsewhere** → `search_for` is
  case-insensitive, so adding `"ÖSTERREICH": "ÖSTERREICH"` also hits
  "Niederlassung **Österreich**" in the corporate footer and re-inserts it
  uppercased on every page. Constrain such keys by position (page + rect) to
  the occurrence you mean.
- **Replacing the bank's own address** → a street regex (`…strasse|gasse|weg|platz`) matches **Gadollaplatz 1** in the letterhead and footer just as happily as the customer's street, and rewrites flatex's Graz address into the synthetic one. Deny-list the corporate boilerplate: `Gadollaplatz 1`, `8010 Graz`, `Große Gallusstr. 16-18`, `60312 Frankfurt am Main`, `Omniturm`.
- **Collateral damage from the redaction rect** → `apply_redactions()` deletes every glyph the rect *touches*, and `search_for` returns a full line box that reaches into the line above. On these pages the recipient name overlaps the small document code printed over it, silently deleting 13 digits of that code. Inset the rect (~18% of its height, plus ~0.3pt horizontally); a glyph is still removed as long as the rect intersects it at all, so shrinking cannot leave PII behind.
- **Losing the punctuation next to a value** → the salutation reads `…<name>,` and the comma sits flush against the name. Redacting just the name clips it, and the extractor's salutation fallback (`Sehr geehrte[rn]? (Herr|Frau) (.+?),`) then matches nothing, so `depot_holder` comes out empty. Claim the comma in the replacement (`"<name>," -> "<new>,"`) and place longer keys first so the bare name does not re-match the same spot.
- **Reusing embedded fonts** → Identity-H subsets can't map new Unicode; insertion silently falls back to Helvetica and the body text stops being monospaced. Always map to the base-14 clone.
- **A `FONTMAP` miss falling through to a default** → the older trade layout names its fonts plain `Courier`/`Helvetica`, which matches none of the `*BFO` keys. With a `return "helv"` default, every replacement is inserted proportional: the page still renders and parses, but each mono column is visibly misaligned. Raise on an unmapped font instead — the failure has to be loud, because the JSON diff cannot see it.
- **Changing digit-string length** → breaks the colon-aligned mono columns. Keep counts equal.
- **Trying to re-insert rotated/vertical codes** → `insert_textbox(..., rotate=90)` often fails to fit and leaves a blank. Leave the barcode/postal codes alone.
- **Trusting Presidio output verbatim** → on German it tags public board members and boilerplate. Curate against the field table.
- **Forgetting the salutation** → `Herrn` left on a female persona. Map it per persona.
