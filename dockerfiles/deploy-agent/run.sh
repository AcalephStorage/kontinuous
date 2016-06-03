#!/bin/bash

setup() {
	mkdir -p /kontinuous/{src,status}/${KONTINUOUS_PIPELINE_ID}/${KONTINUOUS_BUILD_ID}/${KONTINUOUS_STAGE_ID}
}


prepare_kube_config() {
	# replace token for kube config
	sed -i "s/{{token}}/$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)/g" /root/.kube/config
}

wait_for_ready() {
	echo "Waiting for ready signal..."
	until [[ -f /kontinuous/status/${KONTINUOUS_PIPELINE_ID}/${KONTINUOUS_BUILD_ID}/${KONTINUOUS_STAGE_ID}/ready ]]; do
		sleep 5
	done
}


deploy(){
	echo "Deploying App to Kubernetes Cluster"
	if [[ ${DEPLOY_FILES} == "" ]]; then
		echo "Resource/s not found."
		return exit 1
	fi

	i=0
	mkdir -p /tmp/deployfiles

	for fileStr in ${DEPLOY_FILES} 
	do
		echo -n ${fileStr} | base64 -d > /tmp/deployfiles/$i.yml
		let "i+=1"
	done


    kubectl apply -f /tmp/deployfiles
    generate_result $?
}


generate_result(){
	local result="$1"
	if [[ "$result" != "0" ]]; then
			touch /kontinuous/status/${KONTINUOUS_PIPELINE_ID}/${KONTINUOUS_BUILD_ID}/${KONTINUOUS_STAGE_ID}/fail
			echo "Deploy Fail"
			exit 1
		else
			touch /kontinuous/status/${KONTINUOUS_PIPELINE_ID}/${KONTINUOUS_BUILD_ID}/${KONTINUOUS_STAGE_ID}/success
			echo "Deploy Successful"	
			exit 0
	fi
}



main() {
	setup
	prepare_kube_config	
	wait_for_ready
	deploy
}

main $@