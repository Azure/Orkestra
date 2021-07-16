module github.com/Azure/Orkestra

go 1.15

require (
	code.gitea.io/sdk/gitea v0.14.1
	github.com/argoproj/argo-workflows/v3 v3.0.7
	github.com/chartmuseum/helm-push v0.9.0
	github.com/fluxcd/helm-controller/api v0.9.0
	github.com/fluxcd/pkg/apis/meta v0.9.0
	github.com/fluxcd/source-controller/api v0.12.2
	github.com/go-logr/logr v0.4.0
	github.com/gofrs/flock v0.8.0
	github.com/google/go-cmp v0.5.2
	github.com/heptiolabs/healthcheck v0.0.0-20180807145615-6ff867650f40
	github.com/jinzhu/copier v0.3.0
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	go.opencensus.io v0.22.5 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.5.4
	k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v1.4.2-0.20200203170920-46ec8731fbce
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.8
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.8.3
)
