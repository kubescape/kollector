package kubernetes.api.client

# This information could be retrieved from the kubernetes API
# too, but would essentially require a request per API group,
# so for now use a lookup table for the most common resources.
resource_group_mapping := {
	"services": "api/v1",
	"pods": "api/v1",
	"configmaps": "api/v1",
	"secrets": "api/v1",
	"persistentvolumeclaims": "api/v1",
	"daemonsets": "apis/apps/v1",
	"deployments": "apis/apps/v1",
	"statefulsets": "apis/apps/v1",
	"horizontalpodautoscalers": "api/autoscaling/v1",
	"jobs": "apis/batch/v1",
	"cronjobs": "apis/batch/v1beta1",
	"ingresses": "api/extensions/v1beta1",
	"replicasets": "apis/apps/v1",
	"networkpolicies": "apis/networking.k8s.io/v1",
}

# Query for given resource/name in provided namespace
# Example: query_ns("deployments", "my-app", "default")
query_name_ns(resource, name, namespace) = http.send({
	"url": sprintf("https://10.0.2.15:8443/%v/namespaces/%v/%v/%v", [
		resource_group_mapping[resource],
		namespace,
		resource,
		name,
	]),
	"method": "get",
	"headers": {"authorization": sprintf("Bearer %v", ["eyJhbGciOiJSUzI1NiIsImtpZCI6ImhWeW1ZN3pLcGF5T1lYOEtYbFQ4ZTF0QTJUYjlMdEh0Vm94ek5LY1o2VzQifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJjeWJlcmFybW9yLXN5c3RlbSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJjYS1jb250cm9sbGVyLXNlcnZpY2UtYWNjb3VudC10b2tlbi1yajd4eiIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50Lm5hbWUiOiJjYS1jb250cm9sbGVyLXNlcnZpY2UtYWNjb3VudCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6IjgwMjBmNWYxLThjOGMtNDg1NC05YWQ0LWIwOGY1Y2EyNzUyZCIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDpjeWJlcmFybW9yLXN5c3RlbTpjYS1jb250cm9sbGVyLXNlcnZpY2UtYWNjb3VudCJ9.QZw0WcLJ593aL-aR_bmrR8HfuozBPPoxq9bbQAqAsJOOpKhUrVLi3RQ5xhF5HoUVOTPis6EyXvnmTsMc4edFo-IbaY9OS_lp9FRIvyBGqJynaDdUIe55XhEzyLZrHDc33Ver0XYw2L9k9SapCbcDIMiUoRDeGZD0J-gb-wrA9dqRoq_fBKnBRkFmd3EPMNQX-D5cQzeWjfFBNYu2BYJnFP_tGmMpbndCddNpVfYjIbaYN8FS5nDwe5YPDBywIWKiEZZArekOPHFBna2Z6tJWsXU2I1b9YDjKQAwK-yUDEvOACfCj9brWaQ5pcOB8livTwJcZYJIEjeZ-LE8p7mQpSg"])},
	"tls_ca_cert_file": "/home/david/temp/nginx_cert.crt",
	"raise_error": true,
})

# Query for given resource type using label selectors
# https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#api
# Example: query_label_selector_ns("deployments", {"app": "opa-kubernetes-api-client"}, "default")
query_label_selector_ns(resource, selector, namespace) = http.send({
	"url": sprintf("https://kubernetes.default.svc/%v/namespaces/%v/%v?labelSelector=%v", [
		resource_group_mapping[resource],
		namespace,
		resource,
		label_map_to_query_string(selector),
	]),
	"method": "get",
	"headers": {"authorization": sprintf("Bearer %v", [opa.runtime().env.KUBERNETES_API_TOKEN])},
	"tls_ca_cert_file": "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
	"raise_error": false,
})

# Query for all resources of type resource in all namespaces
# Example: query_all("deployments")
query_all(resource) = http.send({
	"url": sprintf("https://kubernetes.default.svc/%v/%v", [
		resource_group_mapping[resource],
		resource,
	]),
	"method": "get",
	"headers": {"authorization": sprintf("Bearer %v", [opa.runtime().env.KUBERNETES_API_TOKEN])},
	"tls_ca_cert_file": "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
	"raise_error": false,
})

label_map_to_query_string(map) = concat(",", [str | val := map[key]; str := concat("%3D", [key, val])])