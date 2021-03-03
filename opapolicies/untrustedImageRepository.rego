package armo_builtins

untrustedImageRepo[msga] {
	k := input.kind
	k == "Pod"
	image := input.spec.containers[_].image
	startswith(image, "015253967648.dkr.ecr.eu-central-1.amazonaws.com/")
	selfLink := input.metadata.selfLink

	msg := sprintf("image '%v' in [%s] comes from untrusted registry",
     [image, selfLink])
	msga := {
		"alert-message": msg,
		"alert": true,
		"prevent": false,
		"alert-score": 3,
		"alert-object": "armo_builtins.untrustedImageRepo"
	}
}
