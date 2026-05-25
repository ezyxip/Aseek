prebuilt-dir := "prebuilt/linux-arm64"
orchestrator-bin := prebuilt-dir + "/aseek-orchestrator"

build-orchestrator:
    mkdir -p {{prebuilt-dir}}
    cd orchestrator && GOARCH=arm64 GOOS=linux CGO_ENABLED=0 \
        go build -o ../{{orchestrator-bin}} -ldflags="-s -w" .
    echo "Built {{orchestrator-bin}}"

clean-orchestrator:
    rm -f {{orchestrator-bin}}