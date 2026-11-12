#!/usr/bin/env bash
# Runs the frontend on this machine against the deployed stack.   or: make web
#
# Only the web page is local. The API, the sign-in, the storage and the
# durable function are the ones in AWS.
set -euo pipefail

. "$(dirname "$0")/lib.sh"
need aws npm

root=$(cd "$(dirname "$0")/.." && pwd)
api=$(stack_output ApiUrl)

cat <<EOF

  Frontend on http://localhost:3000, using the deployed API in $AWS_REGION:
  $api

  Sign in with a user from \`make user NAME=<username>\`.

EOF

cd "$root/frontend"
# An empty local user: a value left in .env.local must not skip the sign-in.
NEXT_PUBLIC_ENDPOINT="$api" NEXT_PUBLIC_LOCAL_USER="" exec npm run dev
