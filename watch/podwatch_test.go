package watch

import (
	"context"
	_ "embed"

	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
)

//go:embed testdata/pod.json
var podJson string

//go:embed testdata/podOD.json
var podOD string

//go:embed testdata/runningPod.json
var runningPodJson string

//go:embed testdata/runningPodOD.json
var runningPodOD string

func TestAddPodScanNotificationCandidateList(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(podOD), &od)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)
	ctx := context.Background()
	addPodScanNotificationCandidateList(ctx, &od, &pod)
	exist, _ := isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")
}

func TestRemovePodScanNotificationCandidateList(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(podOD), &od)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	ctx := context.Background()
	addPodScanNotificationCandidateList(ctx, &od, &pod)
	exist, _ := isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")

	removePodScanNotificationCandidateList(&od, &pod)
	exist, _ = isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.False(t, exist, "pod should not exist")
}

func TestCheckNotificationCandidateList(t *testing.T) {
	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(podOD), &od)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	ctx := context.Background()
	addPodScanNotificationCandidateList(ctx, &od, &pod)
	exist, _ := isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	assert.True(t, checkNotificationCandidateList(&runningPod, &runningOd, "Running"), "pod should be reported")
}

/*
use case:

	use case:	              will add to list:	will reported:   will remove to list:  supported:
	new microservice created	  yes	            yes	            no                    yes
*/
func TestNewMicroserviceCreated(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(podOD), &od)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	ctx := context.Background()
	addPodScanNotificationCandidateList(ctx, &od, &pod)
	exist, _ := isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	assert.True(t, checkNotificationCandidateList(&runningPod, &runningOd, "Running"), "pod should be reported")
}

/*
use case:

	use case:	                                              will add to list:	will reported:   will remove to list:  supported:
	new microservice created with bad configuration or error	  yes	            no	            yes                   yes
*/
func TestNewMicroserviceCreatedWithBadConfiguration(t *testing.T) {

	scanNotificationCandidateList = []*ScanNewImageData{}
	od := OwnerDet{}
	pod := core.Pod{}

	err := json.Unmarshal([]byte(podJson), &pod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(podOD), &od)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	ctx := context.Background()
	addPodScanNotificationCandidateList(ctx, &od, &pod)
	exist, _ := isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	assert.False(t, checkNotificationCandidateList(&runningPod, &runningOd, "Failed"), "pod should not be reported")

	removePodScanNotificationCandidateList(&runningOd, &runningPod)
	exist, _ = isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.False(t, exist, "pod should not exist")
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
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(podOD), &od)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	ctx := context.Background()
	addPodScanNotificationCandidateList(ctx, &od, &pod)
	if exist, _ := isPodAlreadyExistInScanCandidateList(ctx, &od, &pod); !exist {
		t.Fatalf("pod should exist")
	}

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	assert.True(t, checkNotificationCandidateList(&runningPod, &runningOd, "Running"), "pod should be reported")

	addPodScanNotificationCandidateList(ctx, &od, &pod)
	exist, _ := isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")

	runningNewOd := OwnerDet{}
	runningNewPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningNewPod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(runningPodOD), &runningNewOd)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	runningNewPod.Status.ContainerStatuses[0].Image = "nginx:perl"
	runningNewPod.Status.ContainerStatuses[0].ImageID = "nginx:perlImageID"
	assert.True(t, checkNotificationCandidateList(&runningNewPod, &runningNewOd, "Running"), "pod should be reported")

	removePodScanNotificationCandidateList(&od, &pod)
	exist, _ = isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")
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
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(podOD), &od)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	ctx := context.Background()
	addPodScanNotificationCandidateList(ctx, &od, &pod)
	exist, _ := isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	assert.True(t, checkNotificationCandidateList(&runningPod, &runningOd, "Running"), "pod should be reported")

	addPodScanNotificationCandidateList(ctx, &od, &pod)
	exist, _ = isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")

	runningNewOd := OwnerDet{}
	runningNewPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningNewPod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(runningPodOD), &runningNewOd)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	runningNewPod.Status.ContainerStatuses[0].Image = "nginx:perl"
	runningNewPod.Status.ContainerStatuses[0].ImageID = "nginx:perlImageID"
	assert.False(t, checkNotificationCandidateList(&runningNewPod, &runningNewOd, "Failed"), "pod should not be reported")

	removePodScanNotificationCandidateList(&od, &pod)
	exist, _ = isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")
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
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(podOD), &od)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	ctx := context.Background()
	addPodScanNotificationCandidateList(ctx, &od, &pod)
	exist, _ := isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")

	runningOd := OwnerDet{}
	runningPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningPod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(runningPodOD), &runningOd)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	assert.True(t, checkNotificationCandidateList(&runningPod, &runningOd, "Running"), "pod should be reported")

	addPodScanNotificationCandidateList(ctx, &od, &pod)
	exist, _ = isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")

	runningNewOd := OwnerDet{}
	runningNewPod := core.Pod{}

	err = json.Unmarshal([]byte(runningPodJson), &runningNewPod)
	assert.NoErrorf(t, err, "failed convert pod to json: %v", err)

	err = json.Unmarshal([]byte(runningPodOD), &runningNewOd)
	assert.NoErrorf(t, err, "failed convert od to json: %v", err)

	runningNewPod.Spec.Containers[0].ImagePullPolicy = "IfNotPresent"
	assert.False(t, checkNotificationCandidateList(&runningNewPod, &runningNewOd, "Running"), "pod should not be reported")

	removePodScanNotificationCandidateList(&od, &pod)
	exist, _ = isPodAlreadyExistInScanCandidateList(ctx, &od, &pod)
	assert.True(t, exist, "pod should exist")
}
