build: generate
	go build .

generate:
	npx buf generate

docker-build:
	 docker buildx build --build-arg SERVICE_NAME=staff-simulator --build-arg VERSION=1.0.1 -t staff-simulator:1.0.1 --load .
