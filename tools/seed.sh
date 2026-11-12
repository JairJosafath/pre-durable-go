#!/usr/bin/env bash
# Fills a drive with sample photos, so there is something to look at, search
# for and review.
#
#   tools/seed.sh aws <username>     the deployed stack    or: make seed NAME=<username>
#   tools/seed.sh local [user]       the local stack       or: make local-seed
#
# The photos are downloaded once into local/samples/ (see tools/samples.py):
# about twenty random ones, and four with beer or wine that should end up
# with a reviewer. Set FRESH=1 to download a new random set, or SAMPLES=<dir>
# to upload your own JPEGs and PNGs instead.
#
# Uploading the same photo again replaces it: the pipeline runs again.
#
# In AWS every photo is analysed by Amazon Rekognition, which is billed per
# image: the whole set costs about 5 US cents outside the free tier.
set -euo pipefail

. "$(dirname "$0")/lib.sh"

root=$(cd "$(dirname "$0")/.." && pwd)
samples=${SAMPLES:-$root/local/samples}
local_api=${LOCAL_API:-localhost:8080}

usage() {
  echo "usage: $0 aws <username> | local [user]" >&2
  exit 2
}

# photos prints the images in the samples folder, one per line.
photos() {
  find "$samples" -maxdepth 1 -type f \( -iname '*.jpg' -o -iname '*.jpeg' -o -iname '*.png' \) | sort
}

download_samples() {
  if [ -n "${SAMPLES:-}" ]; then
    [ -d "$samples" ] || die "SAMPLES=$samples is not a folder"

    return
  fi

  if [ -n "${FRESH:-}" ]; then
    rm -rf "$samples"
  fi

  if [ ! -f "$samples/SOURCES.txt" ]; then
    echo "Downloading sample photos from Wikimedia Commons ..."
    python3 "$root/tools/samples.py" "$samples"
  fi
}

content_type() {
  case "${1,,}" in
    *.png) echo image/png ;;
    *) echo image/jpeg ;;
  esac
}

# In AWS the photos go straight into the upload bucket, under the user's
# prefix. That is where the browser puts them too, and the object landing
# there is what starts the durable function.
seed_aws() {
  local username=$1 pool subject bucket
  need aws

  pool=$(stack_resource AWS::Cognito::UserPool)
  bucket=$(stack_output UploadBucket)
  subject=$(aws cognito-idp admin-get-user --user-pool-id "$pool" --username "$username" \
    --query "UserAttributes[?Name=='sub'].Value | [0]" --output text 2>/dev/null) || subject=
  found "$subject" || die "no user $username. Create it with: make user NAME=$username"

  aws s3 cp "$samples" "s3://$bucket/users/$subject/" --recursive --no-progress \
    --exclude '*' --include '*.jpg' --include '*.jpeg' --include '*.png' \
    --include '*.JPG' --include '*.JPEG' --include '*.PNG'

  echo
  echo "Uploaded to $username's drive. Every photo started an execution of the durable function."
  echo "Run \`make web\`, sign in as $username, and answer the photos under \"Waiting for your review\"."
}

# Locally the photos go through the local API, which stands in for the
# upload bucket and its S3 event.
seed_local() {
  local user=$1 photo name
  need curl

  curl -s -o /dev/null "http://$local_api/files" ||
    die "nothing answers on http://$local_api. Start the local stack with: make local"

  while read -r photo; do
    name=$(basename "$photo")
    curl -sS --fail -o /dev/null -X PUT -H "Content-Type: $(content_type "$name")" \
      --data-binary "@$photo" "http://$local_api/uploads/users/$user/$name"
    echo "  $name"
  done < <(photos)

  echo
  echo "Uploaded to $user's local drive. Open http://localhost:3000 and answer the photos under \"Waiting for your review\"."
  echo "The local classifier is not Rekognition: expect odd labels, and more photos in review."
}

[ $# -ge 1 ] || usage
target=$1

case "$target" in
  aws) [ $# -eq 2 ] || usage ;;
  local) [ $# -le 2 ] || usage ;;
  *) usage ;;
esac

need python3
download_samples
[ -n "$(photos)" ] || die "no JPEG or PNG in $samples"

case "$target" in
  aws) seed_aws "$2" ;;
  local) seed_local "${2:-alice}" ;;
esac
