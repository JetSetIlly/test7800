version = v0.4.0-preview

goBinary = go
gcflags = -c 3 -B -wb=false -l -l -l -l
ldflags = -s -w
ldflags_version = $(ldflags) -X 'github.com/jetsetilly/test7800/version.number=$(version)'

### support targets
.PHONY: all tidy generate

all:
	@echo "use 'release' to build release binary"

tidy:
# goimports is not part of the standard Go distribution so we won't won't
# require this in any of the other targets
	goimports -w .

generate:
	@$(goBinary) generate ./...

### release building

.PHONY: version_check release

version_check :
ifndef version
	$(error version is undefined)
endif

release: version_check generate 
	$(goBinary) build -pgo=auto -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version)" -tags="release"
	mv test7800 test7800_$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)


### cross compilation for windows (tested when cross compiling from Linux)
.PHONY: cross_windows_release 

cross_windows_release: version_check generate
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "static imguifreetype release" -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version) -H=windowsgui" -o test7800_windows_amd64.exe .

cross_windows_terminal_release: version_check generate
	CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" CXX="/usr/bin/x86_64-w64-mingw32-g++" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -L/usr/local/x86_64-w64-mingw32/lib" $(goBinary) build -pgo=auto -tags "static imguifreetype release" -gcflags "$(gcflags)" -trimpath -ldflags "$(ldflags_version) -H=windows" -o test7800_windows_amd64_terminal.exe .
