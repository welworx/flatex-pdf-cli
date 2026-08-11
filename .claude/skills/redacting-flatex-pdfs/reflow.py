"""Splice appended redaction text back into a page's main content stream.

PyMuPDF's ``insert_text`` / ``add_redact_annot(text=...)`` both *append* the
replacement as a new content stream at the end of the page's ``/Contents``.
The page renders correctly, but extractors that read in content-stream order
(rather than sorting geometrically) see every replaced value drop out of its
slot and reappear as a block of loose text at the bottom of the page.

This module moves each appended text block to its reading-order position inside
the main stream, so stream-order and geometric extraction agree. Rendering is
unchanged: the moved blocks keep their absolute ``Tm`` and their own font, and
the white cover rectangles travel with them.

Usage:
    doc = fitz.open(path)
    reflow(doc)
    doc.save(out, garbage=4, deflate=True)

Anything the parser does not fully understand raises ``ReflowError`` rather
than writing a half-edited stream.
"""

import re

import fitz

__all__ = ["reflow", "reflow_page", "ReflowError"]


class ReflowError(RuntimeError):
    """Raised when a content stream uses constructs this editor cannot rewrite."""


# One appended insert_text block, as PyMuPDF emits it.
_SNIPPET = re.compile(
    r"q\s*BT\s*1 0 0 1 (?P<x>[-\d.]+) (?P<y>[-\d.]+) Tm\s*"
    r"/(?P<font>\w+) (?P<size>[\d.]+) Tf[^\[]*"
    r"\[(?P<arr>[^\]]*)\]TJ\s*ET\s*Q",
    re.S,
)

# One white cover rectangle, as PyMuPDF emits it for a redaction fill.
_RECT = re.compile(
    r"q\s*(?P<x>[-\d.]+) (?P<y>[-\d.]+) (?P<w>[-\d.]+) (?P<h>[-\d.]+) re\s*h\s*"
    r"1 1 1 RG 1 1 1 rg B\s*Q",
    re.S,
)

_NUM = r"[-+]?[\d.]+"
# Operators the run-tracker understands inside a BT ... ET block.
_OPS = re.compile(
    rf"(?P<BT>\bBT\b)"
    rf"|(?P<ET>\bET\b)"
    rf"|(?P<Tf>/(?P<fname>\w+)\s+(?P<fsize>{_NUM})\s+Tf)"
    rf"|(?P<Tm>{_NUM}\s+{_NUM}\s+{_NUM}\s+{_NUM}\s+"
    rf"(?P<mx>{_NUM})\s+(?P<my>{_NUM})\s+Tm)"
    rf"|(?P<Td>(?P<dx>{_NUM})\s+(?P<dy>{_NUM})\s+(?:Td|TD))"
    rf"|(?P<TL>(?P<lead>{_NUM})\s+TL)"
    rf"|(?P<Tstar>\bT\*)"
    rf"|(?P<TJ>\[[^\]]*\]\s*TJ)"
    rf"|(?P<Tj>\((?:[^()\\]|\\.)*\)\s*Tj)"
    rf"|(?P<cm>\bcm\b)"
    rf"|(?P<q>\bq\b)"
    rf"|(?P<Q>\bQ\b)",
)

# A kern this large only appears where redaction removed text (real inter-glyph
# kerns are a few dozen thousandths of an em; these are tens of points).
# Operators that carry state a run-by-run rebuild would silently drop.
_UNSUPPORTED = re.compile(
    r"(?<![A-Za-z0-9])(Tc|Tw|Tz|Ts|Tr|gs|Do|sc|scn|rg|RG|\bg\b|\bG\b|\bk\b|\bK\b)"
    r"(?![A-Za-z0-9])"
)

_BIG_KERN = 1000.0

# How far apart two runs can start and still count as the same column, when
# choosing where a whole-line redaction should fall back to (points).
_COLUMN = 60.0

_KERN_ELEM = re.compile(rf"(?<![\d.])(-\d{{4,}}(?:\.\d+)?)(?![\d.])")

_STRING = re.compile(r"\((?:[^()\\]|\\.)*\)|<[0-9A-Fa-f\s]*>")


def _strip_strings(s):
    """Blank out string literals so operator scans cannot match document text."""
    return _STRING.sub(lambda m: " " * len(m.group(0)), s)


class _Run:
    """One text-showing operator, with its absolute position on the page."""

    __slots__ = ("x", "y", "start", "end", "text", "font", "size", "block")

    def __init__(self, x, y, start, end, text, font, size, block):
        self.x, self.y = x, y
        self.start, self.end = start, end
        self.text = text
        self.font, self.size = font, size
        self.block = block


class _Block:
    """One BT ... ET block."""

    __slots__ = ("start", "end", "runs", "transformed")

    def __init__(self, start, transformed=False):
        self.start, self.end, self.runs = start, None, []
        # True when a cm is active, so this block's text coordinates are not
        # page coordinates. flatex uses this only for the rotated margin
        # barcodes, which are artifacts and never redaction targets.
        self.transformed = transformed


def _scan(stream):
    """Return the BT..ET blocks of ``stream`` with every text run positioned.

    Only the default CTM is tracked; a ``cm`` inside a text-bearing block makes
    positions unreliable, so those blocks are marked and skipped by the caller.
    """
    blocks, block = [], None
    x = y = 0.0          # current text position
    lx = ly = 0.0        # line-start position (what Td is relative to)
    leading = 0.0
    font, size = None, None
    ctm = [False]        # q/Q stack; True once a cm is applied at that level

    # Match operators against a copy with string literals blanked, so document
    # text can never be read as an operator. Blanking preserves length, so all
    # offsets still index the original stream.
    for m in _OPS.finditer(_strip_strings(stream)):
        if m.group("q"):
            ctm.append(ctm[-1])
        elif m.group("Q"):
            if len(ctm) > 1:
                ctm.pop()
        elif m.group("cm"):
            ctm[-1] = True
        elif m.group("BT"):
            block = _Block(m.start(), transformed=ctm[-1])
            x = y = lx = ly = 0.0
        elif m.group("ET"):
            if block is not None:
                block.end = m.end()
                blocks.append(block)
                block = None
        elif m.group("Tf"):
            font, size = m.group("fname"), float(m.group("fsize"))
        elif m.group("Tm"):
            x = lx = float(m.group("mx"))
            y = ly = float(m.group("my"))
        elif m.group("Td"):
            lx += float(m.group("dx"))
            ly += float(m.group("dy"))
            x, y = lx, ly
        elif m.group("TL"):
            leading = float(m.group("lead"))
        elif m.group("Tstar"):
            ly -= leading
            x, y = lx, ly
        elif m.group("TJ") or m.group("Tj"):
            if block is None:
                continue
            block.runs.append(
                _Run(x, y, m.start(), m.end(), stream[m.start() : m.end()],
                     font, size, block)
            )

    if block is not None:
        raise ReflowError("unterminated BT block")
    return blocks


def _tj_array(run_text):
    """The inner text of a ``[...]TJ`` run, or None for a ``(..)Tj`` run."""
    if not run_text.lstrip().startswith("["):
        return None
    return run_text[run_text.index("[") + 1 : run_text.rindex("]")]


def _split_at_kern(arr):
    """Split a TJ array at its redaction gap. Returns (before, after) or None."""
    hits = [m for m in _KERN_ELEM.finditer(_strip_strings(arr))
            if abs(float(m.group(1))) >= _BIG_KERN]
    if len(hits) != 1:
        return None
    m = hits[0]
    return arr[: m.start()], arr[m.end() :]


def _fmt(v):
    return f"{v:.4f}".rstrip("0").rstrip(".")


def _emit_run(run, at_x, at_y, arr=None):
    """A standalone BT..ET drawing ``run`` (or ``arr``) at an absolute position."""
    body = run.text if arr is None else f"[{arr}]TJ"
    font = f"/{run.font} {_fmt(run.size)} Tf " if run.font else ""
    return f"\nBT {font}{_fmt(at_x)} {_fmt(at_y)} Td {body} ET\n"


def reflow_page(doc, page):
    """Move this page's appended redaction text into the main content stream."""
    xrefs = page.get_contents()
    if len(xrefs) < 2:
        return 0

    parts = [doc.xref_stream(x).decode("latin-1") for x in xrefs]
    main, extras = parts[0], parts[1:]

    snippets, rects = [], []
    for part in extras:
        consumed = 0
        for m in _SNIPPET.finditer(part):
            snippets.append(
                {
                    "x": float(m.group("x")),
                    "y": float(m.group("y")),
                    "text": m.group(0),
                }
            )
            consumed += len(m.group(0))
        for m in _RECT.finditer(part):
            rects.append(
                {
                    "x0": float(m.group("x")),
                    "y0": float(m.group("y")),
                    "x1": float(m.group("x")) + float(m.group("w")),
                    "y1": float(m.group("y")) + float(m.group("h")),
                    "text": m.group(0),
                    "used": False,
                }
            )
            consumed += len(m.group(0))
        # Refuse to drop anything that is neither a snippet nor a cover rect.
        if len(part.strip()) - consumed > 40:
            raise ReflowError(
                f"extra content stream holds {len(part.strip()) - consumed} "
                "unrecognised bytes; refusing to discard it"
            )

    if not snippets:
        return 0

    blocks = _scan(main)
    runs = [r for b in blocks if not b.transformed for r in b.runs]
    if not runs:
        raise ReflowError("main stream has no text runs to anchor against")

    # Pair each cover rectangle with the snippet whose baseline it covers, so
    # the two move together and the rect never paints over the text.
    for s in snippets:
        for r in rects:
            if not r["used"] and r["x0"] - 1 <= s["x"] <= r["x1"] and r["y0"] - 1 <= s["y"] <= r["y1"] + 1:
                s["rect"] = r["text"]
                r["used"] = True
                break

    # Plan every move as "attach to run R", either splitting R at its redaction
    # gap or following it. Several snippets can land on one run or one block, so
    # nothing is written until the whole plan is known.
    plan = {}      # id(run) -> {"run": run, "before"/"split"/"after": [...]}
    prepend = []   # snippets that precede all text on the page

    def slot(run):
        return plan.setdefault(
            id(run), {"run": run, "before": [], "split": [], "after": []}
        )

    # Reading order, so several values attaching to one run stack up correctly.
    for s in sorted(snippets, key=lambda s: (-s["y"], s["x"])):
        payload = s["text"] if "rect" not in s else s["rect"] + "\n" + s["text"]
        same_line = [r for r in runs if abs(r.y - s["y"]) < 1.5 and r.x <= s["x"] + 0.1]

        if not same_line:
            # Value starts the line, with the rest of the row to its right
            # (an order row begins with its Auftrags-Nr). Put it ahead of them.
            to_right = [r for r in runs if abs(r.y - s["y"]) < 1.5 and r.x > s["x"] + 0.1]
            if to_right:
                slot(min(to_right, key=lambda r: r.x))["before"].append(payload)
                continue

            # The line is empty, because redaction removed all of it. Fall in
            # after the last run that precedes it, but stay in the same column:
            # these pages put an address block beside a details column, and
            # anchoring across the gutter interleaves the two into one
            # unreadable run of text.
            before = [r for r in runs if (-r.y, r.x) < (-s["y"], s["x"])]
            column = [r for r in before if abs(r.x - s["x"]) < _COLUMN]
            candidates = column or before
            if not candidates:
                prepend.append(payload)
            else:
                slot(max(candidates, key=lambda r: (-r.y, r.x)))["after"].append(payload)
            continue

        run = max(same_line, key=lambda r: r.x)
        arr = _tj_array(run.text)
        split = _split_at_kern(arr) if arr is not None else None
        if split is None:
            # Value sits at the end of the line, so it simply follows the run.
            slot(run)["after"].append(payload)
            continue

        # Value sits mid-line, in the gap the big kern left behind. Record where
        # to cut and where the text after the gap resumes. `None` from
        # _span_x_after means only trailing whitespace follows, which draws no
        # marks wherever it lands.
        after_x = _span_x_after(page, s)
        slot(run)["split"].append((split, payload, s["x"] if after_x is None else after_x))

    # Rebuild each touched block once, emitting every run at an absolute
    # position so no run inherits a line origin shifted by an insertion.
    edits = []
    touched = {}
    for entry in plan.values():
        touched.setdefault(id(entry["run"].block), entry["run"].block)
    for block in touched.values():
        # Rebuilding a block drops any state set inside it, so only rebuild
        # blocks that carry nothing but positioning, font and text operators.
        leftover = _UNSUPPORTED.search(_strip_strings(main[block.start : block.end]))
        if leftover:
            raise ReflowError(
                f"text block uses {leftover.group(0)!r}, which a rebuild "
                "would discard"
            )
        rebuilt = ""
        for run in block.runs:
            entry = plan.get(id(run))
            if entry is None:
                rebuilt += _emit_run(run, run.x, run.y)
                continue
            for payload in entry["before"]:
                rebuilt += "\n" + payload + "\n"
            if entry["split"]:
                if len(entry["split"]) > 1:
                    raise ReflowError("two redaction gaps in one text run")
                (before_arr, after_arr), payload, after_x = entry["split"][0]
                if before_arr.strip():
                    rebuilt += _emit_run(run, run.x, run.y, before_arr)
                rebuilt += "\n" + payload + "\n"
                rebuilt += _emit_run(run, after_x, run.y, after_arr)
            else:
                rebuilt += _emit_run(run, run.x, run.y)
            for payload in entry["after"]:
                rebuilt += "\n" + payload + "\n"
        edits.append((block.start, block.end, rebuilt))

    out = main
    for start, end, text in sorted(edits, key=lambda e: -e[0]):
        out = out[:start] + text + out[end:]
    if prepend:
        out = "\n".join(prepend) + "\n" + out

    doc.update_stream(xrefs[0], out.encode("latin-1"))
    doc.xref_set_key(page.xref, "Contents", f"{xrefs[0]} 0 R")
    return len(snippets)


def _span_x_after(page, snippet):
    """x where the text following a redaction gap begins, from PyMuPDF layout.

    None when nothing visible follows the gap on that line.

    Content-stream coordinates are PDF user space (origin bottom-left) while
    get_text reports PyMuPDF space (origin top-left), so the snippet position is
    converted before comparing.
    """
    want = fitz.Point(snippet["x"], snippet["y"]) * page.transformation_matrix
    best = None
    for block in page.get_text("dict")["blocks"]:
        for line in block.get("lines", []):
            for span in line["spans"]:
                ox, oy = span["origin"]
                if abs(oy - want.y) < 1.5 and ox > want.x + 0.1:
                    if best is None or ox < best:
                        best = ox
    if best is None:
        return None
    return (fitz.Point(best, want.y) * ~page.transformation_matrix).x


def reflow(doc):
    """Reflow every page. Returns the number of blocks moved."""
    return sum(reflow_page(doc, page) for page in doc)


if __name__ == "__main__":
    import sys

    for path in sys.argv[1:]:
        d = fitz.open(path)
        n = reflow(d)
        d.saveIncr() if n else None
        print(f"{path}: moved {n}")
