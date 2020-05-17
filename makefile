linux:
	go build 
	./e621_crawler

mac:
	make linux

windows:
	go build
	./e621_crawler.exe