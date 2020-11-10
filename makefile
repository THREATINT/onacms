all: clean deps amd64

deps:
	go get all

amd64: deps
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -ldflags '-s -w -extldflags "-static"'

arm64: deps
	CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -a -ldflags '-s -w -extldflags "-static"'

upx:
	upx --brute onacms

docker:onacms_x64
	upx --brute onacms
	dos2unix Dockerfile
	docker build -t threatint/onacms .

docker-deploy:docker
	docker push threatint/onacms

clean:
	rm -rf onacms onacms.upx
