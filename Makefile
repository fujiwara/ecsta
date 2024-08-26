.PHONY: test install clean

TNCL_VERSION=v0.0.4

test:
	TZ=Asia/Tokyo go test ./...

install:
	go install github.com/fujiwara/ecsta/cmd/ecsta

dist/:
	goreleaser build --snapshot --rm-dist

clean:
	rm -fr dist/ assets/*

download-assets:
	cd assets && \
	gh release download $(TNCL_VERSION) \
		--repo fujiwara/tncl \
		--pattern 'tncl-*-linux-musl'
