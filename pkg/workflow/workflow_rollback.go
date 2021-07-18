package workflow

import (
	"context"
	"fmt"

	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/Azure/Orkestra/pkg/meta"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (wc *RollbackWorkflowClient) GetLogger() logr.Logger {
	return wc.Logger
}

func (wc *RollbackWorkflowClient) GetClient() client.Client {
	return wc.Client
}

func (wc *RollbackWorkflowClient) GetType() v1alpha1.WorkflowType {
	return v1alpha1.Rollback
}

func (wc *RollbackWorkflowClient) GetNamespace() string {
	return wc.namespace
}

func (wc *RollbackWorkflowClient) GetOptions() ClientOptions {
	return wc.ClientOptions
}

func (wc *RollbackWorkflowClient) GetAppGroup() *v1alpha1.ApplicationGroup {
	return wc.appGroup
}

func (wc *RollbackWorkflowClient) GetWorkflow(ctx context.Context) (*v1alpha13.Workflow, error) {
	rollbackWorkflow := &v1alpha13.Workflow{}
	rollbackWorkflowName := fmt.Sprintf("%s-rollback", wc.appGroup.Name)
	err := wc.Get(ctx, types.NamespacedName{Namespace: wc.namespace, Name: rollbackWorkflowName}, rollbackWorkflow)
	return rollbackWorkflow, err
}

func (wc *RollbackWorkflowClient) Generate(ctx context.Context) error {
	if wc.appGroup == nil {
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	rollbackInstance := wc.appGroup.DeepCopy()
	lastSuccessful := wc.appGroup.GetLastSuccessful()
	if lastSuccessful == nil {
		return meta.ErrPreviousSpecNotSet
	}
	rollbackInstance.Spec = *lastSuccessful
	rollbackWorkflowName := fmt.Sprintf("%s-rollback", rollbackInstance.Name)
	wc.workflow = initWorkflowObject(rollbackWorkflowName, wc.namespace, wc.parallelism)

	entryTemplate, templates, err := generateTemplates(rollbackInstance, wc.GetOptions())
	if err != nil {
		return fmt.Errorf("failed to generate argo workflow: %w", err)
	}
	executorTemplates, err := generateExecutorTemplates(wc.appGroup, Install)
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Update with the app dag templates, entry template, and executor template
	updateWorkflowTemplates(wc.workflow, templates...)
	updateWorkflowTemplates(wc.workflow, *entryTemplate)
	updateWorkflowTemplates(wc.workflow, executorTemplates...)

	return nil
}

func (wc *RollbackWorkflowClient) Submit(ctx context.Context) error {
	// Create the new workflow, only if there is not already a rollback workflow that has been created
	wc.workflow.Labels[OwnershipLabel] = wc.appGroup.Name
	if err := controllerutil.SetControllerReference(wc.appGroup, wc.workflow, wc.Scheme()); err != nil {
		return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
	}
	if err := wc.Create(ctx, wc.workflow); !errors.IsAlreadyExists(err) && err != nil {
		return fmt.Errorf("failed to CREATE argo workflow object: %w", err)
	}
	return nil
}
