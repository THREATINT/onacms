amd64:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -ldflags '-s -w -extldflags "-static"'

arm64:
	CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -a -ldflags '-s -w -extldflags "-static"'

upx:
	upx --lzma onacms

docker: upx
	dos2unix Dockerfile
	docker build -t threatint/onacms .

docker-deploy: docker
	docker push threatint/onacms

deps:
	go get -u all

clean:
	rm -rf onacms onacms.upx
