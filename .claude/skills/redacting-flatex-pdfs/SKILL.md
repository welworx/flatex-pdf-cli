---
name: redacting-flatex-pdfs
description: Use when turning real flatexDEGIRO broker PDFs (Kauf/Verkauf trade confirmations, Fondsertragsausschüttung / Ertragsmitteilung dividend statements) into PII-free test fixtures — replacing customer name, address, Depot/Konto/Transaktion/Auftragsnummer with synthetic values while keeping each document byte-for-byte visually identical to the original.
---

# Redacting flatex PDFs into Test Fixtures

## Overview

Real flatex statements contain a customer's name, address, and account numbers. To use them as test fixtures, replace that PII with **synthetic** values while keeping the page visually identical.

**Core technique:** redact the exact PII text rectangles, then re-insert synthetic text at the same position in a *base-14 font*. The original flatex PDFs use embedded Identity-H fonts (`HerosBFO`, `CursorBFO`) throughout — including PII fields. Base-14 substitutes (Helvetica, Courier) are visually similar but technically distinct; the PII fields will render in a slightly different font than the surrounding document. For parsing purposes this does not matter — positions, sizes, and structure are preserved. Do **not** try to reuse the embedded fonts — they are Identity-H subsets and silently produce wrong glyphs for any character not already in the subset.

Tools: **PyMuPDF (`fitz`)** for redaction, **Presidio** to *identify* PII candidates. No new repo script — run inline.

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

Keep digit-string **lengths equal** to the originals so mono-column alignment is preserved. Umlauts (ä ö ü Ö) and hyphens are fine in both Helvetica and Courier (WinAnsi).

## Reference implementation

`page.search_for(old)` gives exact rects; the covering span gives font + baseline. Redact all rects, `apply_redactions()`, then re-insert.

```python
import fitz
FONTMAP = {"CursorBFO-Regular":"cour","CursorBFO-Bold":"cobo",
           "HerosBFO-Regular":"helv","HerosBFO-Bold":"hebo","OfficinaSans":"helv"}
def b14(f):
    for k,v in FONTMAP.items():
        if k in f: return v
    return "helv"

def redact(src, out, replacements):           # replacements: {old_text: synthetic}
    doc = fitz.open(src)
    for page in doc:
        spans = [(fitz.Rect(s["bbox"]), s["origin"], s["font"], s["size"])
                 for b in page.get_text("dict")["blocks"]
                 for l in b.get("lines", []) for s in l["spans"]]
        def span_at(r):
            c = fitz.Point((r.x0+r.x1)/2, (r.y0+r.y1)/2)
            return next((s for s in spans if s[0].contains(c)), None)
        ins = []
        for old, new in replacements.items():
            for r in page.search_for(old):
                s = span_at(r)
                if not s: continue
                page.add_redact_annot(r, fill=(1,1,1))          # erase original
                ins.append((r.x0, s[1][1], new, b14(s[2]), s[3]))  # x, baseline_y, text, font, size
        page.apply_redactions()
        for x, by, new, fn, sz in ins:
            page.insert_text((x, by), new, fontname=fn, fontsize=sz, color=(0,0,0))
    doc.save(out, garbage=4, deflate=True)
```

## Verify before claiming done

1. **Residual scan** — confirm no original token survives in the text:
   `"".join(p.get_text() for p in fitz.open(out))` must not contain any original name fragment or number.
2. **Visual diff** — render before/after to PNG (`page.get_pixmap(dpi=200)`) and confirm layout + fonts match. Check the address block (Helvetica path) and a mono body line (Courier path) at high DPI.

## Common mistakes

- **Reusing embedded fonts** → Identity-H subsets can't map new Unicode; insertion silently falls back to Helvetica and the body text stops being monospaced. Always map to the base-14 clone.
- **Changing digit-string length** → breaks the colon-aligned mono columns. Keep counts equal.
- **Trying to re-insert rotated/vertical codes** → `insert_textbox(..., rotate=90)` often fails to fit and leaves a blank. Leave the barcode/postal codes alone.
- **Trusting Presidio output verbatim** → on German it tags public board members and boilerplate. Curate against the field table.
- **Forgetting the salutation** → `Herrn` left on a female persona. Map it per persona.
