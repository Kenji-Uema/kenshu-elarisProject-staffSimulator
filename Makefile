build: generate
	go build .

generate:
	npx buf generate

docker-build:
	 docker build --build-arg SERVICE_NAME=staff-simulator --build-arg VERSION=latest -t staff-simulator:latest .