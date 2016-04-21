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

create_dependencies() {
	for dependency in ${DEPENDENCIES}; do
		deploy ${dependency}
	done
}

cleanup_dependencies() {
	for dependency in ${DEPENDENCIES}; do
		clean ${dependency}
	done
}

deploy(){
	echo "Deploying App to Kubernetes Cluster"
    #local deployFile="${WORKING_DIR}/${DEPLOY_FILE}"
    local deployFile="${WORKING_DIR}/$1"

    if [[ ! -f deployFile ]]; then
        echo "Deploy Failed. manifest.yml is unavailable."
        return 1
    fi

    kubectl create -f deployFile
    if [[ "$?" != "0" ]]; then
        echo "Deploy Failed. Unable to deploy app."
        return 1
    fi 
    echo "Deploy Successful"
    return 0
}

clean(){
    echo "Cleaning up"
    local deployFile="${WORKING_DIR}/$1"
    if [[ ! -f deployFile ]]; then
        echo "Clean up Failed. File is unavailable."
        return 1
    fi

    kubectl delete -f deployFile
    if [[ "$?" == "1" ]]; then
        echo "Clean up Failed. Unable to remove app from the cluster."
        exit 1
    fi 
    echo "Clean up Successful"
    return 0
}

run_image() {
	local pod_name="$1"
	# get which node the current job is running on
	local node_name=$(kubectl get pods ${pod_name} -o template --template="{{ .spec.nodeName }}")

	# prepare vars
	local env_vars="`for key in ${ENV_KEYS}; do echo \"       - name: $key\"; echo \"         value: $(eval echo \\$$key)\"; done`"

	# do the sed thingy
	cp /root/pod_template.yml /tmp/pod.yml
	sed -i "s|__POD_NAME__|${pod_name}|g" /tmp/pod.yml
	sed -i "s|__NAMESPACE__|${NAMESPACE}|g" /tmp/pod.yml
	sed -i "s|__NODE_NAME__|${node_name}|g" /tmp/pod.yml
	sed -i "s|__IMAGE__|${IMAGE}|g" /tmp/pod.yml
	sed -i "s|__COMMAND__|${COMMAND}|g" /tmp/pod.yml
	sed -i "s|__ENV_VARS__|${env_vars}|g" /tmp/pod.yml

	kubectl create -f /tmp/pod.yml
}

generate_result(){
	local result="$1"
	if [[ "$result" != "0" ]]; then
			touch /kontinuous/status/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/fail
			echo "Build Fail"
			exit 1
		else
			touch /kontinuous/status/${PIPELINE_ID}/${BUILD_ID}/${STAGE_ID}/success
			echo "Build Successful"	
			exit 0
	fi
}

wait_for_success() {
	local pod_name="$1"
	# poll the pod and pass or fail

	local exit_code_line=""
	until [[ "${exit_code_line}" != "" ]]; do
		sleep 5
		exit_code_line=$(kubectl get pods ${pod_name}-cmd -o yaml | grep exitCode)
	done

	local exit_code=$(echo ${exit_code_line} | awk '{print $2}')

	echo "Command Agent Logs:"
	echo "-------------------"
	# print logs afterwards
	kubectl logs --namespace="${NAMESPACE}" "${pod_name}-cmd"

	return exit_code
}

run_command() {

	# if deployment, deploy() else do the stuff below
	if [[ "$DEPLOY" == "TRUE" ]]; then
		local result=$(deploy "${WORKING_DIR}/${DEPLOY_FILE}")
		generate_result ${result}
	fi 


	# check if dependencies are defined
	if [[ "${DEPENDENCIES}" != "" ]]; then
		create_dependencies
	fi

	# run image as a pod in the same node as this job
	local pod_name=$(kubectl get pods --namespace=${NAMESPACE} --selector="pipeline=${PIPELINE_ID},build=${BUILD_ID},stage=${STAGE_ID}" --no-headers | awk '{print $1}')
	run_image ${pod_name}

	local result=$(wait_for_success "${pod_name}")

	# cleanup
	kubectl delete -f /tmp/pod.yml || true

	# check if dependencies are defined
	if [[ "${DEPENDENCIES}" != "" ]]; then
		cleanup_dependencies
	fi
	
	generate_result ${result}
}

main() {
	setup
	prepare_kube_config
	wait_for_ready
	run_command
}

main $@