package armo_builtins

privilegedContainer[msga] {
	k := input.kind
	k == "Pod"
	container := input.spec.containers[_]
	isPrivileged := container.securityContext.privileged
	isPrivileged == true
	selfLink := input.metadata.selfLink
	containerName := container.name

	msg := sprintf("There is privileged container '%s' in %s", [containerName, selfLink])
	msga := {
		"alert-message": msg,
		"alert": true,
		"prevent": false,
		"alert-score": 3,
		"alert-object": "armo_builtins.privilegedContainer",
	}
}
