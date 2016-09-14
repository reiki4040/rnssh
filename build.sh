#!/bin/sh
VERSION=$(git describe --tags)
HASH=$(git rev-parse --verify HEAD)
GOVERSION=$(go version)

ARCHIVE_INCLUDES_FILES="LICENSE README.md"

function usage() {
  cat <<_EOB
rnssh build script.

  - build rnssh binary.
  - create release archive and show sha256 (for homebrew formula)

[Options]
  -a: create archive for release
  -g: run glide up when build
  -s: show current build version for check
  -q: quiet mode

_EOB
}

function show_build_version() {
  echo $VERSION
}

quiet=""
function msg() {
  test -z "$quiet" && echo $*
}

function err_exit() {
  echo $* >&2
  exit 1
}

function build() {
  local dest_dir=$1

  if [ -n "$glideup" ]; then
    msg "run glide up..."
    if [ -n "$quiet" ]; then
      glide -q up
	else
      glide up
	fi
  fi

  msg "start build rnssh..."
  GOOS="darwin" GOARCH="amd64" go build -o "$dest_dir/rnssh" -ldflags "-X main.version=$VERSION -X main.hash=$HASH -X \"main.goversion=$GOVERSION\""
  msg "finished build rnssh."
}

function create_archive() {
  local work_dir="work"
  local dest_dir="archives"
  local current=$(pwd)
  if [ -z "$current" ]; then
    exit 1
  fi

  mkdir -p $current/$dest_dir

  msg "start darwin/amd64 build and create archive file."

  rnssh_prefix="rnssh-$VERSION-darwin-amd64"
  archive_dir="$current/$work_dir/$rnssh_prefix"
  mkdir -p "$archive_dir"

  # build
  build "$archive_dir"

  # something
  for f in $ARCHIVE_INCLUDES_FILES
  do
    cp -a $current/$f $archive_dir/
  done

  msg "creating archive..."
  cd $current/$work_dir

  local taropt="czvf"
  if [ -n "$quiet" ]; then
  taropt="czf"
  fi
  tar $taropt "$rnssh_prefix".tar.gz "./$rnssh_prefix"

  mv "$rnssh_prefix".tar.gz $current/$dest_dir/
  shasum -a 256 "$current/$dest_dir/$rnssh_prefix.tar.gz"
  msg "finished darwin/amd64 build and create archive file."
  echo ""
}

mode="build"
glideup=""
while getopts ashqu OPT
do
  case $OPT in
    a) mode="archive"
       ;;
    s) show_build_version
       exit 0
       ;;
    u) glideup="1"
       ;;
    h) usage
       exit 0
       ;;
    q) quiet=1
       ;;
    *) echo "unknown option."
       usage
       exit 1
       ;;
  esac
done
shift $((OPTIND - 1))

# run build or archive
case $mode in
  "build")
    build $(pwd)
    ;;
  "archive")
    create_archive
    ;;
  *)
    echo "unknown mode"
    usage
    exit 1
    ;;
esac
