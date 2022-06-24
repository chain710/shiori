IMG ?= shiori:latest
dev:
	go build --tags dev
prod:
	go build
serve:
	$(shell pwd)/shiori serve -p 8080 --disable-auth
build-image:
	docker buildx build --tag ${IMG} --platform=linux/amd64 .