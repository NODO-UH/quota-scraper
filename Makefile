build:
	go build -ldflags="-s -w" -o quota-scraper_x64.bin -i src/main.go

compress: build
	upx --brute quota-scraper_x64.bin
