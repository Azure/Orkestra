package workflow

import (
	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// ExecutorAction defines the set of executor actions which can be performed on a helmrelease object
type ExecutorAction string

const (
	Install ExecutorAction = "install"
	Delete  ExecutorAction = "delete"
)

func generateExecutorTemplates(ag *v1alpha1.ApplicationGroup, action ExecutorAction) ([]v1alpha13.Template, error) {
	if ag == nil {
		return nil, fmt.Errorf("applicationGroup cannot be nil")
	}
	ts := make([]v1alpha13.Template, 0)
	for _, app := range ag.Spec.Applications {
		tname := getExecutorTemplateName(app.Name)
		if action == Delete {
			tname = getReverseExecutorTemplateName(app.Name)
		}
		ts = append(ts, executorTemplateBuilder(tname, app.Spec.Executor, action))
	}

	return ts, nil
}

func executorTemplateBuilder(templateName string, executor *v1alpha1.Executor, action ExecutorAction) v1alpha13.Template {
	if executor == nil {
		executor = defaultExecutor()
	}
	executorArgs := []string{"--spec", "{{inputs.parameters.helmrelease}}", "--action", string(action), "--timeout", "{{inputs.parameters.timeout}}", "--interval", "1s"}
	return v1alpha13.Template{
		Name:               templateName,
		ServiceAccountName: workflowServiceAccountName(),
		Inputs: v1alpha13.Inputs{
			Parameters: []v1alpha13.Parameter{
				{
					Name: HelmReleaseArg,
				},
				{
					Name:    TimeoutArg,
					Default: utils.ToAnyStringPtr(DefaultTimeout),
				},
			},
		},
		Executor: &v1alpha13.ExecutorConfig{
			ServiceAccountName: workflowServiceAccountName(),
		},
		Outputs: v1alpha13.Outputs{},
		Container: &corev1.Container{
			Name:  executor.Name, // TODO: we might need to create a unique name
			Image: fmt.Sprintf("%s:%s", executor.Image, executor.Tag),
			Args:  executorArgs,
		},
	}
}

func defaultExecutor() *v1alpha1.Executor {
	return &v1alpha1.Executor{
		Name:  "executor",
		Image: "azureorkestra/executor",
		Tag:   "v0.4.1",
	}
}

func getExecutorTemplateName(appName string) string {
	return appName + "-helmrelease-executor"
}

func getReverseExecutorTemplateName(appName string) string {
	return appName + "-helmrelease-reverse-executor"
}
