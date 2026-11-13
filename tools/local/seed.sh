#!/usr/bin/env bash
# Fills a drive of the local stack with sample photos, so there is something
# to look at, search for and review.
#
#   tools/local/seed.sh [user]      or: make local-seed
#
# ../samples.sh says where the photos come from. Uploading the same photo
# again replaces it: the pipeline runs again.
#
# For the deployed stack, see ../seed.sh.
set -euo pipefail

. "$(dirname "$0")/../lib.sh"
. "$(dirname "$0")/../samples.sh"

[ $# -le 1 ] || { echo "usage: $0 [user]" >&2; exit 2; }

user=${1:-alice}
api=${LOCAL_API:-localhost:8080}

need curl
download_samples

curl -s -o /dev/null "http://$api/files" ||
  die "nothing answers on http://$api. Start the local stack with: make local"

# The photos go through the local API, which stands in for the upload bucket
# and its S3 event.
while read -r photo; do
  name=$(basename "$photo")
  curl -sS --fail -o /dev/null -X PUT -H "Content-Type: $(content_type "$name")" \
    --data-binary "@$photo" "http://$api/uploads/users/$user/$name"
  echo "  $name"
done < <(photos)

echo
echo "Uploaded to $user's local drive. Open http://localhost:3000 and answer the photos under \"Waiting for your review\"."
echo "The local classifier is not Rekognition: expect odd labels, and more photos in review."
