build64:
	go build -o quota-scraper_x64.bin -i src/main.go

build32:
	GOARCH=386 go build -o quota-scraper_x32.bin -i src/main.go
