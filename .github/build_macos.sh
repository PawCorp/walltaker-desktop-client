PLATFORMS="darwin/amd64"
type setopt >/dev/null 2>&1
SCRIPT_NAME=`basename "$0"`
FAILURES=""
SOURCE_FILE=`echo $@ | sed 's/\.go//'`
CURRENT_DIRECTORY=${PWD##*/}
OUTPUT="walltaker"
mkdir -p walltaker/
for PLATFORM in $PLATFORMS; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    BIN_FILENAME="${OUTPUT}-${GOOS}-${GOARCH}"
    CMD="GOOS=${GOOS} GOARCH=${GOARCH} GO111MODULE=on go build -o walltaker/${BIN_FILENAME} $@"
    echo "${CMD}"
    eval $CMD || FAILURES="${FAILURES} ${PLATFORM}"
done
# eval errors
if [[ "${FAILURES}" != "" ]]; then
    echo ""
    echo "${SCRIPT_NAME} failed on: ${FAILURES}"
    exit 1
fi