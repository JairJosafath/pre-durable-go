#!/usr/bin/env bash
# Empties drives in the deployed stack: the photos, their thumbnails and
# everything the catalog knows about them. The users themselves stay, with
# their passwords.
#
#   tools/empty-drive.sh <username>...     or: make empty NAME=<username>
#   tools/empty-drive.sh --all             or: make empty-all
#
# --all empties the drive of every user, and also stops the durable
# executions that are still open, such as uploads waiting for a review.
#
# Nothing can be brought back: the buckets keep no old versions. The script
# shows what it is about to delete and asks first. DRY=1 only shows; YES=1
# does not ask.
set -euo pipefail

. "$(dirname "$0")/lib.sh"
need aws python3

if [ $# -lt 1 ]; then
  echo "usage: $0 <username>... | --all" >&2
  exit 2
fi

pool=$(stack_resource AWS::Cognito::UserPool)
uploads=$(stack_output UploadBucket)
thumbnails=$(stack_resource_named StorageThumbnails)
table=$(stack_output TableName)
# The output is the ARN of the "live" alias; the function's name is the part before it.
workflow=$(stack_output WorkflowFunction | cut -d: -f7)

everyone=
if [ "$1" = "--all" ]; then
  everyone=yes
  read -ra users <<<"$(aws cognito-idp list-users --user-pool-id "$pool" --query "Users[].Username" --output text)"
else
  users=("$@")
fi

subject_of() {
  local subject
  subject=$(aws cognito-idp admin-get-user --user-pool-id "$pool" --username "$1" \
    --query "UserAttributes[?Name=='sub'].Value | [0]" --output text 2>/dev/null) || subject=
  found "$subject" || die "no user $1"
  printf '%s\n' "$subject"
}

# objects <bucket> <subject> counts the objects under a user's prefix.
objects() {
  aws s3api list-objects-v2 --bucket "$1" --prefix "users/$2/" --query "length(Contents || \`[]\`)" --output text
}

# records <subject> counts the catalog items in a user's partition: file
# records, index entries and labels.
records() {
  aws dynamodb query --table-name "$table" --select COUNT \
    --key-condition-expression "pk = :pk" \
    --expression-attribute-values "{\":pk\": {\"S\": \"user#$1\"}}" --query Count --output text
}

# open_executions prints the ARN of every durable execution that has not
# ended, on every version of the workflow function.
open_executions() {
  local version
  for version in $(aws lambda list-versions-by-function --function-name "$workflow" \
    --query "Versions[?Version!='\$LATEST'].Version" --output text); do
    aws lambda list-durable-executions-by-function --function-name "$workflow" --qualifier "$version" \
      --statuses RUNNING --query "DurableExecutions[].DurableExecutionArn" --output text | tr '\t' '\n'
  done | grep . || true
}

# forget <subject> deletes every catalog item in a user's partition, 25 at a
# time, which is as many as DynamoDB takes in one request.
forget() {
  python3 - "$table" "$1" <<'EOF'
import json
import subprocess
import sys

table, subject = sys.argv[1:]


def aws(*arguments):
    return json.loads(subprocess.run(["aws", *arguments, "--output", "json"], check=True, capture_output=True, text=True).stdout or "{}")


keys = aws(
    "dynamodb", "query", "--table-name", table, "--projection-expression", "pk, sk",
    "--key-condition-expression", "pk = :pk",
    "--expression-attribute-values", json.dumps({":pk": {"S": f"user#{subject}"}}),
)["Items"]

waiting = [{"DeleteRequest": {"Key": key}} for key in keys]
while waiting:
    batch, waiting = waiting[:25], waiting[25:]
    answer = aws("dynamodb", "batch-write-item", "--request-items", json.dumps({table: batch}))
    # What DynamoDB could not take this time goes back in the queue.
    waiting += answer.get("UnprocessedItems", {}).get(table, [])

print(f"  {len(keys)} catalog items")
EOF
}

echo "In $STACK ($AWS_REGION) this deletes, for good:"
echo

subjects=()
for user in "${users[@]}"; do
  subject=$(subject_of "$user")
  subjects+=("$subject")
  printf '  %-12s %s photos, %s thumbnails, %s catalog items\n' "$user" \
    "$(objects "$uploads" "$subject")" "$(objects "$thumbnails" "$subject")" "$(records "$subject")"
done

executions=()
if [ -n "$everyone" ]; then
  mapfile -t executions < <(open_executions)
  echo
  echo "  and stops ${#executions[@]} durable execution(s) that are still open"
fi

echo

if [ -n "${DRY:-}" ]; then
  echo "DRY is set: nothing was deleted."
  exit 0
fi

if [ -z "${YES:-}" ]; then
  read -rp 'Type "delete" to go on: ' answer
  [ "$answer" = "delete" ] || die "nothing was deleted"
fi

# Executions first: one that is still running could write to the catalog
# again after its records are gone.
for execution in "${executions[@]}"; do
  aws lambda stop-durable-execution --durable-execution-arn "$execution" >/dev/null
  echo "stopped ${execution##*/}"
done

for index in "${!users[@]}"; do
  subject=${subjects[$index]}
  echo "${users[$index]}:"
  aws s3 rm "s3://$uploads/users/$subject/" --recursive --only-show-errors
  aws s3 rm "s3://$thumbnails/users/$subject/" --recursive --only-show-errors
  echo "  photos and thumbnails"
  forget "$subject"
done

echo
echo "Done. Fill a drive again with: make seed NAME=<username>"
