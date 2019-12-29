onacms:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-s -w -extldflags "-static"'

docker:onacms
	upx --brute onacms
	dos2unix Dockerfile
	docker build -t threatint/onacms .

docker-deploy:docker
	docker push threatint/onacms
