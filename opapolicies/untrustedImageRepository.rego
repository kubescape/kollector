package armo_builtins

untrustedImageRepo[msga] {
	k := input.kind
	k == "Pod"
	container := input.spec.containers[_]
	image := container.image
	startswith(image, "015253967648.dkr.ecr.eu-central-1.amazonaws.com/")
	selfLink := input.metadata.selfLink
	containerName := container.name
	msg := sprintf("image '%v' in container '%s' in [%s] comes from untrusted registry", [image, containerName, selfLink])

	msga := {
		"alert-message": msg,
		"alert": true,
		"prevent": false,
		"alert-score": 3,
		"alert-object": "armo_builtins.untrustedImageRepo",
	}
}
