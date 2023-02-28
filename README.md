# Kollector
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubescape%2Fkollector.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkubescape%2Fkollector?ref=badge_shield)

The Kollector component is an in-cluster component of the [Kubescape security platform](https://www.armosec.io/blog/kubescape-open-source-kubernetes-security-platform/?utm_source=github&utm_medium=repository).  
It communicates with the Kubernetes API server to collect cluster information and watches for changes in the cluster.
The information sent to the Kubescape SaaS platform.

## Building Kollector
To build the kollector run: `go build .`  

## Configuration
Load config file using the `CONFIG` environment variable   

`export CONFIG=path/to/clusterData.json`  

<details><summary>example/clusterData.json</summary>

```json5 
{
   "gatewayWebsocketURL": "127.0.0.1:8001",
   "gatewayRestURL": "127.0.0.1:8002",
   "kubevulnURL": "127.0.0.1:8081",
   "kubescapeURL": "127.0.0.1:8080",
   "eventReceiverRestURL": "https://report.armo.cloud",
   "eventReceiverWebsocketURL": "wss://report.armo.cloud",
   "rootGatewayURL": "wss://ens.euprod1.cyberarmorsoft.com/v1/waitfornotification",
   "accountID": "*********************",
   "clusterName": "******" 
  } 
``` 
</details>

## Environment Variables

Check out `watch/environmentvariables.go`

* `WAIT_BEFORE_REPORT`: Wait before sending the report to the gateway. Default: 60 seconds. This value is in seconds.

## VS code configuration samples

You can use the sample file below to setup your VS code environment for building and debugging purposes.

<details><summary>.vscode/launch.json</summary>

```json5
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Package",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program":  "${workspaceRoot}",
                 "env": {
                     "NAMESPACE": "kubescape",
                     "CONFIG": "${workspaceRoot}/.vscode/clusterData.json",
            },
            "args": [
                "-alsologtostderr", "-v=4", "2>&1"
            ]
        }
    ]
}
```
</details>


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubescape%2Fkollector.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkubescape%2Fkollector?ref=badge_large)