build:
	rm -rf ./bin/*
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/shinobi-proxy-x86_64 .
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/shinobi-proxy-arm_64 .
