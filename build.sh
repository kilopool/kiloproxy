go mod tidy
rm -r ./build
mkdir ./build
cd ./build

echo Building for Windows...
export CGO_ENABLED="0"
export GOOS="windows"
export GOARCH="amd64"
go build ../
zip ./kiloproxy-windows-x64.zip ./kiloproxy.exe

echo Building for Linux...
export CGO_ENABLED="0"
export GOOS="linux"
export GOARCH="amd64"
go build ../
mv ./kiloproxy ./kiloproxy-linux-x64
xz -9 -e ./kiloproxy-linux-x64

echo Building for MacOS/Darwin...
export CGO_ENABLED="0"
export GOOS="darwin"
export GOARCH="amd64"
go build ../
mv ./kiloproxy ./kiloproxy-macos-darwin-x64
xz -9 -e ./kiloproxy-macos-darwin-x64

rm ./kiloproxy.exe

echo Done.

sha256sum *