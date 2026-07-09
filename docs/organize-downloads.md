# Organize Downloads

Part of [flatex-pdf-cli](../README.md). Automatically sort flatex PDFs from
your Downloads folder into a structured archive, one folder per depot, files
renamed by date and document type. Requires `jq` (`brew install jq` on macOS).

## One-time paste

Edit the `TARGET` line, then paste the whole block into your terminal:

```bash
TARGET=~/Documents/flatex-organized
find ~/Downloads -name '*.pdf' | while IFS= read -r pdf; do
  json=$(flatex-pdf-cli -include-metadata -quiet "$pdf" 2>/dev/null) || continue
  account=$(jq -r '.metadata.depot_number // "unknown"' <<<"$json")
  date=$(jq -r '.transactions[0].date' <<<"$json")
  type=$(jq -r '.transactions[0].document_type' <<<"$json")
  dest="$TARGET/$account"
  mkdir -p "$dest"
  cp "$pdf" "$dest/${date}_${type}_$(basename "$pdf")"
  echo "  -> $dest/${date}_${type}_$(basename "$pdf")"
done
```

## Reusable shell function

Add this to your `~/.zshrc` or `~/.bashrc` to call it by name:

```bash
flatex-organize() {
  local src="${1:-$HOME/Downloads}"
  local target="${2:?Usage: flatex-organize [source_dir] <target_dir>}"

  find "$src" -name '*.pdf' | while IFS= read -r pdf; do
    json=$(flatex-pdf-cli -include-metadata -quiet "$pdf" 2>/dev/null) || continue
    account=$(jq -r '.metadata.depot_number // "unknown"' <<<"$json")
    date=$(jq -r '.transactions[0].date' <<<"$json")
    type=$(jq -r '.transactions[0].document_type' <<<"$json")
    dest="$target/$account"
    mkdir -p "$dest"
    cp "$pdf" "$dest/${date}_${type}_$(basename "$pdf")"
    echo "  -> $dest/${date}_${type}_$(basename "$pdf")"
  done
}
```

```bash
flatex-organize ~/Documents/flatex-organized            # source defaults to ~/Downloads
flatex-organize ~/Downloads ~/Documents/flatex-organized
```

## Result layout

```
flatex-organized/
  31022213792/
    2025-09-16_TRADE_20250916_KaufFondsZertifikate_31022213792_517614092.pdf
    2025-10-02_DIVIDEND_20251002_Fondsertragsausschuettung_31022213792_528846930.pdf
```

Non-flatex PDFs in the source directory are silently skipped.
