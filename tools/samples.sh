# The sample photos both seed scripts upload. Source it after lib.sh; do not
# run it.
#
# The photos are downloaded once into tools/samples/ (see samples.py): about
# twenty random ones, and four with beer or wine, most of which end up with a
# reviewer. Set FRESH=1 to download a new random set, or SAMPLES=<dir> to
# upload your own JPEGs and PNGs instead.

tools=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
samples=${SAMPLES:-$tools/samples}

# photos prints the images in the samples folder, one per line.
photos() {
  find "$samples" -maxdepth 1 -type f \( -iname '*.jpg' -o -iname '*.jpeg' -o -iname '*.png' \) | sort
}

# download_samples makes sure there are photos to upload, or stops.
download_samples() {
  if [ -n "${SAMPLES:-}" ]; then
    [ -d "$samples" ] || die "SAMPLES=$samples is not a folder"
  else
    need python3

    if [ -n "${FRESH:-}" ]; then
      rm -rf "$samples"
    fi

    if [ ! -f "$samples/SOURCES.txt" ]; then
      echo "Downloading sample photos from Wikimedia Commons ..."
      python3 "$tools/samples.py" "$samples"
    fi
  fi

  [ -n "$(photos)" ] || die "no JPEG or PNG in $samples"
}

content_type() {
  case "${1,,}" in
    *.png) echo image/png ;;
    *) echo image/jpeg ;;
  esac
}
