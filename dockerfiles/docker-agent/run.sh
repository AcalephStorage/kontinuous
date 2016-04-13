#!/usr/bin/env sh

setup() {
	mkdir -p /kontinuous/{src,status}/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}
}

wait_for_ready() {
	echo "Waiting for ready signal..."
	until [[ -f /kontinuous/status/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/ready ]]; do
		sleep 5
	done
}

build_image() {
	echo "Building docker image..."
	docker build -f /kontinuous/src/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/${DOCKERFILE_PATH}/${DOCKERFILE_NAME} -t ${IMAGE_NAME}:${IMAGE_TAG} /kontinuous/src/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/${DOCKERFILE_PATH}
}
	
push_internal() {
	echo "Pushing Image to local registry: ${IMAGE_NAME}:${IMAGE_TAG} ${INTERNAL_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
	docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${INTERNAL_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}
	docker push ${INTERNAL_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}
}

pull_internal() {
	echo "Pulling Image"
	docker pull ${INTERNAL_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}
}

push_external() {
	echo "Pushing Image"

	eval "local username=\$${USERNAME}"
	eval "local password=\$${PASSWORD}"
	eval "local email=\$${EMAIL}"

	# credentials here? 
	if [[ "$REQUIRE_CREDENTIALS" == "TRUE" ]]; then
		docker login --username=${username} --password=${password} --email=${email} ${EXTERNAL_REGISTRY}
	fi
	docker tag -f ${INTERNAL_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG} ${EXTERNAL_REGISTRY}/${EXTERNAL_IMAGE_NAME}:${IMAGE_TAG}
	docker tag -f ${INTERNAL_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG} ${EXTERNAL_REGISTRY}/${EXTERNAL_IMAGE_NAME}:${BRANCH}
	docker tag -f ${INTERNAL_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG} ${EXTERNAL_REGISTRY}/${EXTERNAL_IMAGE_NAME}:latest
	docker push ${EXTERNAL_REGISTRY}/${EXTERNAL_IMAGE_NAME}:${IMAGE_TAG}
	docker push ${EXTERNAL_REGISTRY}/${EXTERNAL_IMAGE_NAME}:${BRANCH}
	docker push ${EXTERNAL_REGISTRY}/${EXTERNAL_IMAGE_NAME}:latest
}

fail() {
	touch /kontinuous/status/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/fail
	echo "Build Fail"
	exit 1
}

pass() {
	echo "${INTERNAL_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}" > /kontinuous/status/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/docker-image
	touch /kontinuous/status/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/success
	echo "Build Successful"
	exit 0
}

main() {
	setup
	wait_for_ready
	if [[ "${MODE}" == "BUILD" ]]; then
		build_image;   if [[ "$?" != "0" ]]; then fail; fi
		push_internal; if [[ "$?" != "0" ]]; then fail; fi
		pass
	fi

	if [[ "${MODE}" == "PUBLISH" ]]; then
		pull_internal; if [[ "$?" != "0" ]]; then fail; fi
		push_external; if [[ "$?" != "0" ]]; then fail; fi
		pass
	fi
}

main $@