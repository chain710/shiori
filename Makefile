dev:
	go build --tags dev
prod:
	go build
serve:
	$(shell pwd)/shiori serve -p 8080 --disable-auth