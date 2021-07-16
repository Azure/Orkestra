package workflow

import (
	"context"
	"fmt"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"

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

	wc.rollbackAppGroup = wc.appGroup.DeepCopy()
	lastSuccessful := wc.appGroup.GetLastSuccessful()
	if lastSuccessful == nil {
		return meta.ErrPreviousSpecNotSet
	}
	wc.rollbackAppGroup.Spec = *lastSuccessful
	rollbackWorkflowName := fmt.Sprintf("%s-rollback", wc.rollbackAppGroup.Name)
	wc.workflow = initWorkflowObject(rollbackWorkflowName, wc.namespace, wc.parallelism)

	entryTemplate, templates, err := generateTemplates(wc.rollbackAppGroup, wc.GetOptions())
	if err != nil {
		return fmt.Errorf("failed to generate argo workflow: %w", err)
	}
	// Update with the app dag templates, entry template, and executor template
	updateWorkflowTemplates(wc.workflow, templates...)
	updateWorkflowTemplates(wc.workflow, *entryTemplate, wc.executor(HelmReleaseExecutorName, Install))

	return nil
}

func (wc *RollbackWorkflowClient) Submit(ctx context.Context) error {
	if err := wc.purgeNewerReleases(ctx); err != nil {
		return fmt.Errorf("failed to purge helm releases: %w", err)
	}

	// Create the new workflow, only if there is not already a rollback workflow that has been created
	wc.workflow.Labels[v1alpha1.OwnershipLabel] = wc.appGroup.Name
	if err := controllerutil.SetControllerReference(wc.appGroup, wc.workflow, wc.Scheme()); err != nil {
		return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
	}
	if err := wc.Create(ctx, wc.workflow); !errors.IsAlreadyExists(err) && err != nil {
		return fmt.Errorf("failed to CREATE argo workflow object: %w", err)
	}
	return nil
}

// TODO: Purge releases should remove the applications with dependencies in mind
func (wc *RollbackWorkflowClient) purgeNewerReleases(ctx context.Context) error {
	// Get the helm releases that have been deployed at this generation
	diff := wc.getDiff()
	for _, name := range diff {
		listOptions := client.MatchingLabels{
			v1alpha1.OwnershipLabel: wc.appGroup.Name,
			v1alpha1.HeritageLabel:  v1alpha1.HeritageValue,
			v1alpha1.ChartLabel:     name,
		}
		releaseList := &fluxhelmv2beta1.HelmReleaseList{}
		if err := wc.List(ctx, releaseList, listOptions); err != nil {
			return fmt.Errorf("failed to list helm releases deployed by the application group: %w", err)
		}
		for _, release := range releaseList.Items {
			if err := wc.Delete(ctx, &release); err != nil {
				return fmt.Errorf("failed to delete release %s while purging new releases from the lastest rollout: %w", release.Name, err)
			}
		}
	}
	return nil
}

func (wc *RollbackWorkflowClient) getDiff() []string {
	names := wc.appGroup.GetApplicationNames()
	for _, appName := range wc.rollbackAppGroup.GetApplicationNames() {
		names = utils.RemoveStringFromSlice(appName, names)
	}
	return names
}
