package opapoliciesstore

var reportStructBytes = []byte(`
{
    "sessionID": "0000000000000000000000000ae12ef0",
    "createdTime": 101432266588160,
    "transactionID": 13579,
    "responseFormat": "BSON",
    "eventsList": [
        {
            "type": 0,
            "payload": {
                "customerGUID": "1231BCB149CE4A67BDD35DA7A393AE08",
                "solutionGUID": "0807060504030201AAAABBBBBBBBBBBB",
                "componentGUID": "FFFFFFFF00000000FFFFFFFFFFFFFFFF",
                "customerName": "Not specified",
                "solutionName": "Uni-Tests",
                "componentName": "caa_tests",
                "solutionDescription": "One Test To Run Them, One Test to find them all ",
                "componentDescription": "Bugs be aware from the big bad wolf",
                "solutionOwner": "Cyber ArmorSoft Inc.",
                "machineID": "localhost.localdomain",
                "agentVersion": "No agent",
                "trace": "No agent",
                "buildTime": "2021-03-03T15:16:36Z"
            }
        },
        {
            "type": 4096,
            "payload": {
                "osArchitecture": 9,
                "osRelease": "alpine"
            }
        },
        {
            "type": 4097,
            "payload": {
                "machineID": "localhost.localdomain",
                "osProcessID": 1,
                "osProcessName": "/k8s-ca-dashboard-aggregator",
                "processPriority": 0,
                "ASLR": 1,
                "osCGroup": 0,
                "stackProtection": 0,
                "osUserAccount": "root",
                "accountPrivileges": "User"
            }
        },
        {
            "type": 4101,
            "payload": {
                "wlid": "wlid://cluster-k8s-geriatrix-k8s-demo3/namespace-whisky-app/deployment-whisky4all-shipping",
                "groupingLevel0": "k8s-geriatrix-k8s-demo3",
                "groupingLevel1": "cyberarmor-system",
                "namespace": "cyberarmor-system",
                "kind": "Deployment",
                "name": "ca-dashboard-aggregator",
                "cluster": "k8s-geriatrix-k8s-demo3",
                "process_name": "/k8s-ca-dashboard-aggregator",
                "component_level": "container",
                "imageHash": "docker-pullable://quay.io/armosec/k8s-ca-dashboard-aggregator-ubi@sha256:754f3cfca915a07ed10655a301dd7a8dc5526a06f9bd06e7c932f4d4108a8296",
                "imageTag": "quay.io/armosec/k8s-ca-dashboard-aggregator-ubi:latest",
                "container_name": "ca-aggregator",
                "creationDate": "2020-03-03T08:45:34.790867Z",
                "lastEdited": "",
                "signingProfileName": "",
                "autoAccessTokenUpdate": false
            }
        }        
    ]
}
`)
