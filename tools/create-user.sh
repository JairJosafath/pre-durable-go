#!/usr/bin/env bash
# Creates a PrimeDrive user in the deployed stack and prints a password to
# sign in with. Nothing is emailed and there is no code to confirm.
#
#   tools/create-user.sh <username> [email]        or: make user NAME=<username>
#
# The password is generated here. Run the script again for the same user to
# get a new one.
#
# The script needs AWS credentials that may administer the stack's Cognito
# user pool; the credentials that deployed the stack can. Without those, a
# user signs up through POST /auth/signup and POST /auth/confirm, with a code
# sent to their email.
set -euo pipefail

. "$(dirname "$0")/lib.sh"
need aws curl python3

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
  echo "usage: $0 <username> [email]" >&2
  exit 2
fi

username=$1
email=${2:-$username@example.com}

# Four groups joined by underscores: long enough, easy to read out, and one
# double-click selects all of it. It has the upper-case letter, lower-case
# letter, digit and symbol the user pool asks for. Look-alike characters
# (0 O, 1 l I) are left out.
generate_password() {
  python3 -c '
import secrets

letters = "abcdefghijkmnpqrstuvwxyz"
alphabet = letters + letters.upper().replace("I", "").replace("O", "") + "23456789"

while True:
    groups = ["".join(secrets.choice(alphabet) for _ in range(4)) for _ in range(4)]
    password = "_".join(groups)
    if any(c.islower() for c in password) and any(c.isupper() for c in password) and any(c.isdigit() for c in password):
        print(password)
        break
'
}

# json key value [key value ...] prints a JSON object, with every value quoted safely.
json() {
  python3 -c 'import json, sys; print(json.dumps(dict(zip(sys.argv[1::2], sys.argv[2::2]))))' "$@"
}

pool=$(stack_resource AWS::Cognito::UserPool)
api=$(stack_output ApiUrl)
password=$(generate_password)

if aws cognito-idp admin-get-user --user-pool-id "$pool" --username "$username" >/dev/null 2>&1; then
  echo "User $username already exists: giving it a new password."
else
  # The same attributes POST /auth/signup sets, with the email already verified.
  aws cognito-idp admin-create-user --user-pool-id "$pool" --username "$username" \
    --message-action SUPPRESS \
    --user-attributes "Name=email,Value=$email" Name=email_verified,Value=true Name=custom:subscription,Value=free \
    >/dev/null
fi

# A permanent password: the user is not asked to change it at the first sign-in.
aws cognito-idp admin-set-user-password --user-pool-id "$pool" --username "$username" \
  --password "$password" --permanent

# Sign in once the way the frontend does, to know the account really works.
status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "${api%/}/auth/login" \
  -H 'Content-Type: application/json' -d "$(json username "$username" password "$password")")
[ "$status" = 200 ] || die "the user was created, but signing in through ${api%/}/auth/login answered HTTP $status"

cat <<EOF

User $username can sign in.

  Username  $username
  Password  $password

Start the frontend with \`make web\` and sign in at http://localhost:3000.
Give the drive some photos with: make seed NAME=$username
EOF
