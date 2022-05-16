package watch

import (
	"encoding/json"
	"testing"

	core "k8s.io/api/core/v1"
)

var podJson string = "{\"metadata\":{\"name\":\"nginx-6799fc88d8-s6548\",\"generateName\":\"nginx-6799fc88d8-\",\"namespace\":\"default\",\"uid\":\"9080f8b8-d79f-430e-9115-f3fe97920ed2\",\"resourceVersion\":\"35582\",\"creationTimestamp\":\"2022-05-15T11:40:42Z\",\"labels\":{\"app\":\"nginx\",\"pod-template-hash\":\"6799fc88d8\"},\"ownerReferences\":[{\"apiVersion\":\"apps/v1\",\"kind\":\"ReplicaSet\",\"name\":\"nginx-6799fc88d8\",\"uid\":\"4c7f11d1-321d-4363-9c6e-8842eb4d295f\",\"controller\":true,\"blockOwnerDeletion\":true}]},\"spec\":{\"volumes\":[{\"name\":\"default-token-hk8bb\",\"secret\":{\"secretName\":\"default-token-hk8bb\",\"defaultMode\":420}}],\"containers\":[{\"name\":\"nginx\",\"image\":\"nginx\",\"resources\":{},\"volumeMounts\":[{\"name\":\"default-token-hk8bb\",\"readOnly\":true,\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\"}],\"terminationMessagePath\":\"/dev/termination-log\",\"terminationMessagePolicy\":\"File\",\"imagePullPolicy\":\"Always\"}],\"restartPolicy\":\"Always\",\"terminationGracePeriodSeconds\":30,\"dnsPolicy\":\"ClusterFirst\",\"serviceAccountName\":\"default\",\"serviceAccount\":\"default\",\"securityContext\":{},\"schedulerName\":\"default-scheduler\",\"tolerations\":[{\"key\":\"node.kubernetes.io/not-ready\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":300},{\"key\":\"node.kubernetes.io/unreachable\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":300}],\"priority\":0,\"enableServiceLinks\":true,\"preemptionPolicy\":\"PreemptLowerPriority\"},\"status\":{\"phase\":\"Pending\",\"qosClass\":\"BestEffort\"}}"

var podOD string = "{\"name\":\"nginx\",\"kind\":\"Deployment\",\"ownerData\":{\"kind\":\"Deployment\",\"apiVersion\":\"apps/v1\",\"metadata\":{\"name\":\"nginx\",\"namespace\":\"default\",\"uid\":\"56b51636-ba78-4b11-8d2d-5b538d4272c6\",\"resourceVersion\":\"35586\",\"generation\":1,\"creationTimestamp\":\"2022-05-15T11:40:42Z\",\"labels\":{\"app\":\"nginx\"},\"annotations\":{\"deployment.kubernetes.io/revision\":\"1\"}},\"spec\":{\"replicas\":1,\"selector\":{\"matchLabels\":{\"app\":\"nginx\"}},\"template\":{\"metadata\":{\"creationTimestamp\":null,\"labels\":{\"app\":\"nginx\"}},\"spec\":{\"containers\":[{\"name\":\"nginx\",\"image\":\"nginx\",\"resources\":{},\"terminationMessagePath\":\"/dev/termination-log\",\"terminationMessagePolicy\":\"File\",\"imagePullPolicy\":\"Always\"}],\"restartPolicy\":\"Always\",\"terminationGracePeriodSeconds\":30,\"dnsPolicy\":\"ClusterFirst\",\"securityContext\":{},\"schedulerName\":\"default-scheduler\"}},\"strategy\":{\"type\":\"RollingUpdate\",\"rollingUpdate\":{\"maxUnavailable\":\"25%\",\"maxSurge\":\"25%\"}},\"revisionHistoryLimit\":10,\"progressDeadlineSeconds\":600},\"status\":{\"observedGeneration\":1,\"unavailableReplicas\":1,\"conditions\":[{\"type\":\"Progressing\",\"status\":\"True\",\"lastUpdateTime\":\"2022-05-15T11:40:42Z\",\"lastTransitionTime\":\"2022-05-15T11:40:42Z\",\"reason\":\"NewReplicaSetCreated\",\"message\":\"Created new replica set nginx-6799fc88d8\"},{\"type\":\"Available\",\"status\":\"False\",\"lastUpdateTime\":\"2022-05-15T11:40:43Z\",\"lastTransitionTime\":\"2022-05-15T11:40:43Z\",\"reason\":\"MinimumReplicasUnavailable\",\"message\":\"Deployment does not have minimum availability.\"}]}}}"

var runningPodJson string = "{\"metadata\":{\"name\":\"nginx-6799fc88d8-pxgzz\",\"generateName\":\"nginx-6799fc88d8-\",\"namespace\":\"default\",\"uid\":\"f5b3d4ba-24ff-46c1-ab93-b52af6b9b9bd\",\"resourceVersion\":\"35746\",\"creationTimestamp\":\"2022-05-15T11:43:25Z\",\"labels\":{\"app\":\"nginx\",\"pod-template-hash\":\"6799fc88d8\"},\"ownerReferences\":[{\"apiVersion\":\"apps/v1\",\"kind\":\"ReplicaSet\",\"name\":\"nginx-6799fc88d8\",\"uid\":\"0c3c7600-5c17-4d66-9411-3e7f752b90b8\",\"controller\":true,\"blockOwnerDeletion\":true}]},\"spec\":{\"volumes\":[{\"name\":\"default-token-hk8bb\",\"secret\":{\"secretName\":\"default-token-hk8bb\",\"defaultMode\":420}}],\"containers\":[{\"name\":\"nginx\",\"image\":\"nginx\",\"resources\":{},\"volumeMounts\":[{\"name\":\"default-token-hk8bb\",\"readOnly\":true,\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\"}],\"terminationMessagePath\":\"/dev/termination-log\",\"terminationMessagePolicy\":\"File\",\"imagePullPolicy\":\"Always\"}],\"restartPolicy\":\"Always\",\"terminationGracePeriodSeconds\":30,\"dnsPolicy\":\"ClusterFirst\",\"serviceAccountName\":\"default\",\"serviceAccount\":\"default\",\"nodeName\":\"raziel-virtualbox\",\"securityContext\":{},\"schedulerName\":\"default-scheduler\",\"tolerations\":[{\"key\":\"node.kubernetes.io/not-ready\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":300},{\"key\":\"node.kubernetes.io/unreachable\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":300}],\"priority\":0,\"enableServiceLinks\":true,\"preemptionPolicy\":\"PreemptLowerPriority\"},\"status\":{\"phase\":\"Running\",\"conditions\":[{\"type\":\"Initialized\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2022-05-15T11:43:25Z\"},{\"type\":\"Ready\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2022-05-15T11:43:30Z\"},{\"type\":\"ContainersReady\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2022-05-15T11:43:30Z\"},{\"type\":\"PodScheduled\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2022-05-15T11:43:25Z\"}],\"hostIP\":\"10.0.2.15\",\"podIP\":\"172.17.0.6\",\"podIPs\":[{\"ip\":\"172.17.0.6\"}],\"startTime\":\"2022-05-15T11:43:25Z\",\"containerStatuses\":[{\"name\":\"nginx\",\"state\":{\"running\":{\"startedAt\":\"2022-05-15T11:43:30Z\"}},\"lastState\":{},\"ready\":true,\"restartCount\":0,\"image\":\"nginx:latest\",\"imageID\":\"docker-pullable://nginx@sha256:19da26bd6ef0468ac8ef5c03f01ce1569a4dbfb82d4d7b7ffbd7aed16ad3eb46\",\"containerID\":\"docker://de3d7bcf4aa3bfc2e33ad1d7d8e749b1fd7e36f700e8885b566c906827401f83\",\"started\":true}],\"qosClass\":\"BestEffort\"}}"

var runningPodOD string = "{\"name\":\"nginx\",\"kind\":\"Deployment\",\"ownerData\":{\"kind\":\"Deployment\",\"apiVersion\":\"apps/v1\",\"metadata\":{\"name\":\"nginx\",\"namespace\":\"default\",\"uid\":\"155f59e9-3166-41f6-bf16-7b53a5ed9435\",\"resourceVersion\":\"35748\",\"generation\":1,\"creationTimestamp\":\"2022-05-15T11:43:25Z\",\"labels\":{\"app\":\"nginx\"},\"annotations\":{\"deployment.kubernetes.io/revision\":\"1\"}},\"spec\":{\"replicas\":1,\"selector\":{\"matchLabels\":{\"app\":\"nginx\"}},\"template\":{\"metadata\":{\"creationTimestamp\":null,\"labels\":{\"app\":\"nginx\"}},\"spec\":{\"containers\":[{\"name\":\"nginx\",\"image\":\"nginx\",\"resources\":{},\"terminationMessagePath\":\"/dev/termination-log\",\"terminationMessagePolicy\":\"File\",\"imagePullPolicy\":\"Always\"}],\"restartPolicy\":\"Always\",\"terminationGracePeriodSeconds\":30,\"dnsPolicy\":\"ClusterFirst\",\"securityContext\":{},\"schedulerName\":\"default-scheduler\"}},\"strategy\":{\"type\":\"RollingUpdate\",\"rollingUpdate\":{\"maxUnavailable\":\"25%\",\"maxSurge\":\"25%\"}},\"revisionHistoryLimit\":10,\"progressDeadlineSeconds\":600},\"status\":{\"observedGeneration\":1,\"replicas\":1,\"updatedReplicas\":1,\"readyReplicas\":1,\"availableReplicas\":1,\"conditions\":[{\"type\":\"Available\",\"status\":\"True\",\"lastUpdateTime\":\"2022-05-15T11:43:30Z\",\"lastTransitionTime\":\"2022-05-15T11:43:30Z\",\"reason\":\"MinimumReplicasAvailable\",\"message\":\"Deployment has minimum availability.\"},{\"type\":\"Progressing\",\"status\":\"True\",\"lastUpdateTime\":\"2022-05-15T11:43:30Z\",\"lastTransitionTime\":\"2022-05-15T11:43:25Z\",\"reason\":\"NewReplicaSetAvailable\",\"message\":\"ReplicaSet nginx-6799fc88d8 has successfully progressed.\"}]}}}"

func TestAddPodScanNotificationCandidateList(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(podOD), &od)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

}

func TestRemovePodScanNotificationCandidateList(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(podOD), &od)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	removePodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); exist {
		t.Fatalf("pod should not exist")
	}

}

func TestCheckNotificationCandidateList(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(podOD), &od)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	if !checkNotificationCandidateList(&runningPod, &runningOd, "Running") {
		t.Fatalf("pod should reported")
	}
}

/*
use case:
	use case:	              will add to list:	wil reported:   will remove to list:  supported:
	new microsevice created	  yes	            yes	            no                    yes
*/
func TestNewNicroserviceCreated(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(podOD), &od)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	if !checkNotificationCandidateList(&runningPod, &runningOd, "Running") {
		t.Fatalf("pod should reported")
	}
}

/*
use case:
	use case:	                                              will add to list:	wil reported:   will remove to list:  supported:
	new microsevice created with bad configuration or error	  yes	            no	            yes                   yes
*/
func TestNewNicroserviceCreatedWithBadConfiguration(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(podOD), &od)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	if checkNotificationCandidateList(&runningPod, &runningOd, "Failed") {
		t.Fatalf("pod should reported")
	}

	removePodScanNotificationCandidateList(&runningOd, &runningPod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); exist {
		t.Fatalf("pod should not exist")
	}

}

/*
use case:
	use case:	                     will add to list:	               wil reported:   will remove to list:                supported:
	microservice changed it's image  no(the pod counter will increase) yes	           no(the pod counter will decrease)   yes
*/
func TestMicroserviceChangedItsImage(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(podOD), &od)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	if !checkNotificationCandidateList(&runningPod, &runningOd, "Running") {
		t.Fatalf("pod should reported")
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	runningNewOd := OwnerDet{}
	runningNewPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningNewPod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(runningPodOD), &runningNewOd)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	runningNewPod.Status.ContainerStatuses[0].Image = "nginx:perl"
	runningNewPod.Status.ContainerStatuses[0].ImageID = "nginx:perlImageID"
	if !checkNotificationCandidateList(&runningNewPod, &runningNewOd, "Running") {
		t.Fatalf("pod should reported")
	}

	removePodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

}

/*
use case:
	use case:	                                              will add to list:	                will reported:   will remove to list:                supported:
	microservice changed it's image with bad config or error  no(the pod counter will increase) no               no(the pod counter will decrease)   yes
*/
func TestMicroserviceChangedItsImageWithBadConfig(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(podOD), &od)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	if !checkNotificationCandidateList(&runningPod, &runningOd, "Running") {
		t.Fatalf("pod should reported")
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	runningNewOd := OwnerDet{}
	runningNewPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningNewPod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(runningPodOD), &runningNewOd)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	runningNewPod.Status.ContainerStatuses[0].Image = "nginx:perl"
	runningNewPod.Status.ContainerStatuses[0].ImageID = "nginx:perlImageID"
	if checkNotificationCandidateList(&runningNewPod, &runningNewOd, "Failed") {
		t.Fatalf("pod should not reported")
	}

	removePodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

}

/*
use case:
	use case:	                                              will add to list:	                will reported:   will remove to list:                supported:
	microservice changed it's something but the image tag     no(the pod counter will increase) no               no(the pod counter will decrease)   yes
*/
func TestMicroserviceChangedSomethingButItsImage(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(podOD), &od)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	if !checkNotificationCandidateList(&runningPod, &runningOd, "Running") {
		t.Fatalf("pod should reported")
	}

	addPodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	runningNewOd := OwnerDet{}
	runningNewPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningNewPod)
	if err != nil {
		t.Fatalf("failed convert pod to json: %v", err)
	}

	err = json.Unmarshal([]byte(runningPodOD), &runningNewOd)
	if err != nil {
		t.Fatalf("failed convert od to json: %v", err)
	}

	runningNewPod.Spec.Containers[0].ImagePullPolicy = "IfNotPresent"
	if checkNotificationCandidateList(&runningNewPod, &runningNewOd, "Running") {
		t.Fatalf("pod should not reported")
	}

	removePodScanNotificationCandidateList(&od, &pod)
	if exist, _ := isPodAlreadexistInScanCandidateList(&od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

}
