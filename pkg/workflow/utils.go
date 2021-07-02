package workflow

import (
	"os"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getTimeout(t *metav1.Duration) *v1alpha13.AnyString {
	if t == nil {
		return utils.ToAnyStringPtr(DefaultTimeout)
	}
	tm := utils.ToAnyString(t.Duration.String())
	return &tm
}

func GetNamespace() string {
	if ns, ok := os.LookupEnv("WORKFLOW_NAMESPACE"); ok {
		return ns
	}
	return "orkestra"
}

func workflowServiceAccountName() string {
	if sa, ok := os.LookupEnv("WORKFLOW_SERVICEACCOUNT_NAME"); ok {
		return sa
	}
	return "orkestra"
}
