#!/usr/bin/env bash
# Creates a PrimeDrive user through the deployed API: signs up, confirms with
# the code Cognito emails, then logs in once to check the account works.
#
#   tools/create-user.sh <username> <email>
#
# The API URL is read from the PrimeDriveStack outputs in $AWS_REGION
# (default eu-west-1); set PRIMEDRIVE_API to use another deployment.
# The password must have at least 8 characters with an upper-case letter,
# a lower-case letter, a digit and a symbol.
set -euo pipefail

if [ $# -ne 2 ]; then
  echo "usage: $0 <username> <email>" >&2
  exit 2
fi

username=$1
email=$2
region=${AWS_REGION:-eu-west-1}

api=${PRIMEDRIVE_API:-$(aws cloudformation describe-stacks --region "$region" --stack-name PrimeDriveStack \
  --query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" --output text)}
api=${api%/}

# json key value [key value ...] prints a JSON object, with every value quoted safely.
json() {
  python3 -c 'import json, sys; print(json.dumps(dict(zip(sys.argv[1::2], sys.argv[2::2]))))' "$@"
}

# post <path> <json body> prints the response body and fails on an HTTP error.
post() {
  local out status
  out=$(curl -sS -X POST "$api$1" -H 'Content-Type: application/json' -d "$2" -w '\n%{http_code}')
  status=${out##*$'\n'}
  out=${out%$'\n'*}
  if [ "$status" -ge 300 ]; then
    echo "POST $1 failed with HTTP $status: $out" >&2
    return 1
  fi
  printf '%s\n' "$out"
}

read -rsp "Password for $username: " password
echo

echo "Signing up $username at $api ..."
post /auth/signup "$(json username "$username" password "$password" email "$email")" >/dev/null

read -rp "Verification code sent to $email: " code
post /auth/confirm "$(json username "$username" code "$code")" >/dev/null

post /auth/login "$(json username "$username" password "$password")" | grep -q AuthenticationResult
echo "User $username is confirmed and can log in."
