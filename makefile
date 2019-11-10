onacms: onacms.go
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-s -w -extldflags "-static"'
	upx --brute onacms

docker:onacms
	docker build -t threatint/onacms .

docker-deploy:docker
	docker push threatint/onacms
