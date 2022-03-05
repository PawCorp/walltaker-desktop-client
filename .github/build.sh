PLATFORMS="darwin/amd64"
PLATFORMS="$PLATFORMS windows/amd64"
PLATFORMS="$PLATFORMS linux/amd64 linux/386"
PLATFORMS_ARM="linux"
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
    if [[ "${GOOS}" == "windows" ]]; then BIN_FILENAME="${BIN_FILENAME}.exe"; fi
    CMD="GOOS=${GOOS} GOARCH=${GOARCH} go build -o walltaker/${BIN_FILENAME} $@"
    echo "${CMD}"
    eval $CMD || FAILURES="${FAILURES} ${PLATFORM}"
done
# ARM builds
if [[ $PLATFORMS_ARM == *"linux"* ]]; then 
    CMD="GOOS=linux GOARCH=arm64 go build -o walltaker/${OUTPUT}-linux-arm64 $@"
    echo "${CMD}"
    eval $CMD || FAILURES="${FAILURES} ${PLATFORM}"
fi
for GOOS in $PLATFORMS_ARM; do
    GOARCH="arm"
    for GOARM in 7; do
    BIN_FILENAME="${OUTPUT}-${GOOS}-${GOARCH}${GOARM}"
    CMD="GOARM=${GOARM} GOOS=${GOOS} GOARCH=${GOARCH} go build -o walltaker/${BIN_FILENAME} $@"
    echo "${CMD}"
    eval "${CMD}" || FAILURES="${FAILURES} ${GOOS}/${GOARCH}${GOARM}" 
    done
done
# eval errors
if [[ "${FAILURES}" != "" ]]; then
    echo ""
    echo "${SCRIPT_NAME} failed on: ${FAILURES}"
    exit 1
fi