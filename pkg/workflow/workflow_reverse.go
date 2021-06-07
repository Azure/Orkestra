package workflow

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (engine ReverseEngine) GetLogger() logr.Logger {
	return engine.Logger
}

func (engine ReverseEngine) Generate(ctx context.Context) error {
	if engine.forwardWorkflow == nil {
		engine.Error(nil, "forward workflow object cannot be nil")
		return fmt.Errorf("forward workflow object cannot be nil")
	}

	engine.reverseWorkflow = initWorkflowObject(engine.parallelism)

	// Set name and namespace based on the forward workflow
	engine.reverseWorkflow.Name = fmt.Sprintf("%s-reverse", engine.forwardWorkflow.Name)
	engine.reverseWorkflow.Namespace = workflowNamespace()

	entry, err := generateWorkflow(ctx, engine, engine.nodes, engine.forwardWorkflow)
	if err != nil {
		engine.Error(err, "failed to generate reverse workflow")
		return fmt.Errorf("failed to generate argo reverse workflow : %w", err)
	}

	updateWorkflowTemplates(engine.reverseWorkflow, *entry)
	updateWorkflowTemplates(engine.reverseWorkflow, defaultExecutor(HelmReleaseReverseExecutorName, Delete))

	return nil
}

func (engine ReverseEngine) Submit(ctx context.Context) error {
	if engine.reverseWorkflow == nil {
		engine.Error(nil, "reverse workflow object cannot be nil")
		return fmt.Errorf("reverse workflow object cannot be nil")
	}

	if engine.forwardWorkflow == nil {
		engine.Error(nil, "forward workflow object cannot be nil")
		return fmt.Errorf("forward workflow object cannot be nil")
	}

	obj := &v1alpha12.Workflow{}

	err := engine.Get(ctx, types.NamespacedName{Namespace: engine.reverseWorkflow.Namespace, Name: engine.reverseWorkflow.Name}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			// Add OwnershipReference
			err = controllerutil.SetControllerReference(engine.forwardWorkflow, engine.reverseWorkflow, engine.Scheme())
			if err != nil {
				engine.Error(err, "unable to set forward workflow as owner of Argo reverse Workflow object")
				return fmt.Errorf("unable to set forward workflow as owner of Argo reverse Workflow: %w", err)
			}

			// If the argo Workflow object is NotFound and not AlreadyExists on the cluster
			// create a new object and submit it to the cluster
			err = engine.Create(ctx, engine.reverseWorkflow)
			if err != nil {
				engine.Error(err, "failed to CREATE argo workflow object")
				return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
			}
		} else {
			engine.Error(err, "failed to GET workflow object with an unrecoverable error")
			return fmt.Errorf("failed to GET workflow object with an unrecoverable error : %w", err)
		}
	}

	return nil
}

func generateWorkflow(ctx context.Context, l logr.Logger, nodes map[string]v1alpha12.NodeStatus, forward *v1alpha12.Workflow) (*v1alpha12.Template, error) {
	graph, err := Build(forward.Name, nodes)
	if err != nil {
		l.Error(err, "failed to build the wf status DAG")
		return nil, fmt.Errorf("failed to build the wf status DAG : %w", err)
	}

	rev := graph.Reverse()

	entry := &v1alpha12.Template{
		Name: EntrypointTemplateName,
		DAG: &v1alpha12.DAGTemplate{
			Tasks: make([]v1alpha12.DAGTask, 0),
		},
	}

	var prevbucket []fluxhelmv2beta1.HelmRelease
	for _, bucket := range rev {
		for _, hr := range bucket {
			task := v1alpha12.DAGTask{
				Name:     utils.ConvertToDNS1123(hr.GetReleaseName() + "-" + hr.Namespace),
				Template: HelmReleaseReverseExecutorName,
				Arguments: v1alpha12.Arguments{
					Parameters: []v1alpha12.Parameter{
						{
							Name:  HelmReleaseArg,
							Value: utils.ToStrPtr(base64.StdEncoding.EncodeToString([]byte(utils.HrToYaml(hr)))),
						},
					},
				},
				Dependencies: utils.ConvertSliceToDNS1123(getTaskNamesFromHelmReleases(prevbucket)),
			}

			entry.DAG.Tasks = append(entry.DAG.Tasks, task)
		}
		prevbucket = bucket
	}

	if len(entry.DAG.Tasks) == 0 {
		return nil, fmt.Errorf("entry template must have at least one task")
	}

	return entry, nil
}