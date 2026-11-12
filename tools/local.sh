#!/usr/bin/env bash
# Runs the whole of PrimeDrive on this machine in one terminal: the local API
# in the background and the frontend in front of it.        or: make local
#
# The containers (object store, database, classifier) must be up already;
# `make local` starts them first. Ctrl+C stops the API and the frontend and
# leaves the containers running, so the next start is quick.
set -euo pipefail

. "$(dirname "$0")/lib.sh"
need go npm curl

root=$(cd "$(dirname "$0")/.." && pwd)
api=${LOCAL_API:-localhost:8080}
user=${LOCAL_USER:-alice}

cat "$root/local/NOTICE.txt"

if curl -s -o /dev/null "http://$api/files"; then
  die "something already answers on http://$api. Stop it first (is \`make local-api\` running in another terminal?)"
fi

# A built binary, not `go run`: its process is the server itself, so stopping
# it on the way out really stops the server.
mkdir -p "$root/bin"
(cd "$root" && go build -o bin/local ./cmd/local)

# The API's lines are marked, so they can be told apart from the frontend's.
"$root/bin/local" -serve "$api" > >(sed -u 's/^/[api] /') 2>&1 &
server=$!
trap 'kill "$server" 2>/dev/null || true' EXIT

for _ in $(seq 1 60); do
  kill -0 "$server" 2>/dev/null || die "the local API stopped. Are the containers up? Try: make local-up"
  curl -s -o /dev/null "http://$api/files" && break
  sleep 0.5
done

curl -s -o /dev/null "http://$api/files" || die "the local API did not answer on http://$api within 30 seconds"

cat <<EOF
  Open http://localhost:3000. You are signed in as $user.
  In another terminal, \`make local-seed\` uploads sample photos.
  Ctrl+C stops the API and the frontend; \`make local-down\` stops the containers.

EOF

cd "$root/frontend"
NEXT_PUBLIC_ENDPOINT="http://$api" NEXT_PUBLIC_LOCAL_USER="$user" npm run dev
