#!/usr/bin/env bash
# Fills a drive of the deployed stack with sample photos, so there is
# something to look at, search for and review.
#
#   tools/seed.sh <username>        or: make seed NAME=<username>
#
# samples.sh says where the photos come from. Uploading the same photo again
# replaces it: the pipeline runs again.
#
# The photos are uploaded one every two seconds, because a new account allows
# only 10 Lambda executions at once. PACE=<seconds> changes the pause; PACE=0
# is fine for an account with the usual limit of 1,000.
#
# Every photo is analysed by Amazon Rekognition, which is billed per image:
# the whole set costs about 5 US cents outside the free tier.
#
# For the local stack, see local/seed.sh.
set -euo pipefail

. "$(dirname "$0")/lib.sh"
. "$(dirname "$0")/samples.sh"

[ $# -eq 1 ] || { echo "usage: $0 <username>" >&2; exit 2; }

username=$1
pace=${PACE:-2}

need aws
download_samples

pool=$(stack_resource AWS::Cognito::UserPool)
bucket=$(stack_output UploadBucket)
subject=$(aws cognito-idp admin-get-user --user-pool-id "$pool" --username "$username" \
  --query "UserAttributes[?Name=='sub'].Value | [0]" --output text 2>/dev/null) || subject=
found "$subject" || die "no user $username. Create it with: make user NAME=$username"

# The photos go straight into the upload bucket, under the user's prefix.
# That is where the browser puts them too, and the object landing there is
# what starts the durable function.
#
# One photo at a time, with a pause in between. Every upload runs the durable
# function and the thumbnailer at the same moment, and a new AWS account
# allows only 10 Lambda executions at once. Uploaded in one burst, most of
# the set ends as FAILED: the thumbnailer is refused with "Rate Exceeded".
echo "Uploading one photo every $pace seconds, to stay under Lambda's concurrency limit ..."

while read -r photo; do
  name=$(basename "$photo")
  aws s3 cp "$photo" "s3://$bucket/users/$subject/$name" --only-show-errors
  echo "  $name"
  sleep "$pace"
done < <(photos)

echo
echo "Uploaded to $username's drive. Every photo started an execution of the durable function."
echo "Run \`make web\`, sign in as $username, and answer the photos under \"Waiting for your review\"."
