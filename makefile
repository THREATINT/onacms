onacms: onacms.go
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-s -w -extldflags "-static"'

docker:onacms
	docker build -t threatint/onacms .
	upx --brute onacms

docker-deploy:docker
	docker push threatint/onacms
