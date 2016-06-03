#!/bin/bash -e

TAG=${TAG:-latest}
IMAGES="kontinuous-agent docker-agent command-agent deploy-agent"

echo_plus() {
	local message=$1
	echo '================================================================================'
    echo ${message}
	echo '================================================================================'
}

for image in ${IMAGES}; do
	echo_plus "Building ${image}..."
	docker build -t local/${image}:${TAG} ${image}
	echo_plus "Deploying ${image}..."
	docker tag -f local/${image}:${TAG} quay.io/acaleph/${image}:${TAG}
	docker tag -f local/${image}:${TAG} quay.io/acaleph/${image}:latest
	docker push quay.io/acaleph/${image}:${TAG}
	docker push quay.io/acaleph/${image}:latest
done
