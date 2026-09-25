VERSION ?= 0.1.0
OUT     ?= dist

.PHONY: build build-arm64 build-amd64 clean

build: build-arm64 build-amd64

build-arm64:
	@mkdir -p $(OUT)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
		go build -buildmode=c-shared -o $(OUT)/bps-usage-v$(VERSION)-linux-arm64.so .
	@rm -f $(OUT)/*.h

build-amd64:
	@mkdir -p $(OUT)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build -buildmode=c-shared -o $(OUT)/bps-usage-v$(VERSION)-linux-amd64.so .
	@rm -f $(OUT)/*.h

clean:
	rm -rf $(OUT)
