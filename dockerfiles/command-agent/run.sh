#!/bin/bash

setup() {
	mkdir -p /kontinuous/{src,status}/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}
}

prepare_kube_config() {
	# replace token for kube config
	sed -i "s/{{token}}/$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)/g" /root/.kube/config
}

wait_for_ready() {
	echo "Waiting for ready signal..."
	until [[ -f /kontinuous/status/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/ready ]]; do
		sleep 5
	done
}

run_command() {
	echo "Running command..."
	local pod_name=$(kubectl get pods --namespace=${NAMESPACE} --selector="pipeline=${PIPELINE_ID},build=${BUILD_ID},stage=${STAGE_ID}" --no-headers | awk '{print $1}')

	kubectl exec --namespace=${NAMESPACE} ${pod_name} ${CONTAINER_NAME} -- ${COMMAND}
	if [[ "$?" != "0" ]]; then
		touch /kontinuous/status/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/fail
		echo "Build Fail"
		exit 1
	fi
	touch /kontinuous/status/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/success
	echo "Build Successful"
	exit 0
}

main() {
	setup
	prepare_kube_config
	wait_for_ready
	run_command
}

main $@