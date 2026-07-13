#!/usr/bin/env bash
# Prune the generated go/v4 bindings down to the exchanges actually used by
# Somnia consumers. The upstream module is one giant Go package (~110
# exchanges, ~34 MB of generated source); a cold compile takes 20-30 min on a
# 2-core CI runner. Keeping only the exchanges we use cuts that to well under
# a minute, everywhere (CI, docker, local, -race).
#
# Re-run after every upstream sync, then rebuild:
#   go/prune-exchanges.sh binance okx bybit kucoin gate gateio mexc kraken
#   (cd go/v4 && go build ./... && go vet ./...)
#
# Note: some exchanges embed another one (e.g. gateio embeds gate) — the base
# must be kept too or the build breaks, which `go build` will surface.
set -euo pipefail

[[ $# -ge 1 ]] || {
  echo "usage: $0 <exchange-id>..." >&2
  exit 1
}
cd "$(dirname "$0")/v4"

KEEPLIST="$*"
is_kept() {
  local x
  for x in $KEEPLIST; do [[ "$x" == "$1" ]] && return 0; done
  return 1
}

deleted=0
for w in *_wrapper.go; do
  ex="${w%_wrapper.go}"
  is_kept "$ex" && continue
  rm -f "${ex}.go" "${ex}_api.go" "$w"
  deleted=$((deleted + 1))
done

pro_deleted=0
for w in pro/*_wrapper.go; do
  ex="$(basename "$w" _wrapper.go)"
  is_kept "$ex" && continue
  rm -f "pro/${ex}.go" "$w"
  pro_deleted=$((pro_deleted + 1))
done

# Excise pruned exchanges from the generated registries. Case bodies in these
# switches never contain nested case/default clauses, and each switch closes
# with a brace at case-indent, so a line-oriented state machine is safe.
prune_switch() {
  local file="$1"
  awk -v keeplist="$KEEPLIST" '
    BEGIN { n = split(keeplist, a, " "); for (i = 1; i <= n; i++) K[a[i]] = 1 }
    {
      if ($0 ~ /^[ \t]*case "/) {
        id = $0
        sub(/^[ \t]*case "/, "", id)
        sub(/".*/, "", id)
        lid = tolower(id)
        skip = !(lid in K) && lid != "exchange"
      } else if ($0 ~ /^[ \t]*default:/ || $0 ~ /^(\t|    )\}/) {
        skip = 0
      }
      if (!skip) print
    }
  ' "$file" >"$file.pruned"
  mv "$file.pruned" "$file"
}

# Filter the generated `var Exchanges []string` id list down to kept ids,
# preserving the original order and the line's surrounding declaration.
prune_list() {
  local file="$1"
  awk -v keeplist="$KEEPLIST" '
    BEGIN { n = split(keeplist, a, " "); for (i = 1; i <= n; i++) K[a[i]] = 1 }
    /^var Exchanges \[\]string/ {
      out = "var Exchanges []string = []string{"
      m = split($0, parts, "\"")
      first = 1
      for (i = 2; i <= m; i += 2) {
        if (parts[i] in K) {
          out = out (first ? "" : ", ") "\"" parts[i] "\""
          first = 0
        }
      }
      print out "}"
      next
    }
    { print }
  ' "$file" >"$file.pruned"
  mv "$file.pruned" "$file"
}

for f in exchange_dynamic.go exchange_typed_interface.go pro/exchange_dynamic.go pro/exchange_typed_interface.go; do
  prune_switch "$f"
done
for f in exchange_metadata.go pro/exchange_metadata.go; do
  prune_list "$f"
done

echo "kept: $KEEPLIST"
echo "deleted: ${deleted} exchanges from v4, ${pro_deleted} from v4/pro"
echo "remaining v4 wrappers: $(ls ./*_wrapper.go | wc -l | tr -d ' ')"
echo "remaining pro wrappers: $(ls pro/*_wrapper.go | wc -l | tr -d ' ')"
