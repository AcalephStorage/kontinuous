package kube

import "strings"

func (k *realKubeClient) GetPodNameBySelector(namespace string, selector map[string]string) (string, error) {
	a := make([]string, 0)
	for l, v := range selector {
		a = append(a, l+"="+v)
	}
	labelSelector := strings.Join(a, ",")

	uri := "/api/v1/namespaces/" + namespace + "/pods?labelSelector=" + labelSelector
	var response map[string]interface{}
	err := k.doGet(uri, &response)
	if err != nil {
		return "", err
	}

	pod := response["items"].([]interface{})[0].(map[string]interface{})
	podName := pod["metadata"].(map[string]interface{})["name"].(string)

	return podName, nil
}

func (k *realKubeClient) GetPodContainers(namespace, podName string) ([]string, error) {
	uri := "/api/v1/namespaces/" + namespace + "/pods/" + podName
	var response map[string]interface{}
	err := k.doGet(uri, response)
	if err != nil {
		return nil, err
	}

	containers := make([]string, 0)
	cons := response["spec"].(map[string]interface{})["containers"].([]interface{})
	for _, c := range cons {
		cname := c.(map[string]interface{})["name"].(string)
		containers = append(containers, cname)
	}

	return containers, nil
}

func (k *realKubeClient) GetLog(namespace, pod, container string) (string, error) {
	uri := "/api/v1/namespaces/" + namespace + "/pods/" + pod + "/log"
	if container != "" {
		uri = uri + "?container=" + container
	}
	var response []byte
	err := k.doGet(uri, &response)
	if err != nil {
		return "", err
	}
	return string(response), nil
}
