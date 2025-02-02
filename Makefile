build:
	rm bin/shinobi-proxy
	go build -ldflags="-s -w" -trimpath -o bin/shinobi-proxy .
