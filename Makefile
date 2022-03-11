.PHONY: test install clean

test:
	go test ./...

install:
	go install github.com/fujiwara/ecsta/cmd/ecsta

dist/:
	goreleaser build --snapshot --rm-dist

clean:
	rm -fr dist/
