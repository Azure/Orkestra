package helpers

import (
	"context"
	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	"github.com/Azure/Orkestra/pkg/workflow"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusHelper struct {
	client.Client
	logr.Logger
	PatchFrom             client.Patch
	WorkflowClientBuilder *workflow.Builder
	Recorder              record.EventRecorder
}

func (helper *StatusHelper) UpdateStatus(ctx context.Context, instance *v1alpha1.ApplicationGroup) (ctrl.Result, error) {
	chartConditionMap, subChartConditionMap, err := helper.marshallChartStatus(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	instance.Status.Applications = getAppStatus(instance, chartConditionMap, subChartConditionMap)

	// update the workflow status
	result, err := helper.updateWorkflowStatus(ctx, instance)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (helper *StatusHelper) updateWorkflowStatus(ctx context.Context, instance *v1alpha1.ApplicationGroup) (ctrl.Result, error) {
	forwardClient := helper.WorkflowClientBuilder.Forward(instance).Build()
	reverseClient := helper.WorkflowClientBuilder.Reverse(instance).Build()
	rollbackClient := helper.WorkflowClientBuilder.Rollback(instance).Build()

	for _, wfClient := range []workflow.Client{forwardClient, reverseClient, rollbackClient} {
		if err := workflow.UpdateStatus(ctx, wfClient); err != nil {
			return ctrl.Result{}, err
		}
	}
	if isFailed, err := workflow.IsFailed(ctx, forwardClient); err != nil {
		return ctrl.Result{}, err
	} else if isFailed {
		// TODO: make this error come from the node itself
		helper.MarkFailed(instance, fmt.Errorf("workflow in failed state"))
		return ctrl.Result{RequeueAfter: v1alpha1.GetInterval(instance)}, nil
	}
	if isSucceeded, err := workflow.IsSucceeded(ctx, forwardClient); err != nil {
		return ctrl.Result{}, err
	} else if isSucceeded {
		if err := helper.MarkSucceeded(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: v1alpha1.GetInterval(instance)}, nil
	}
	return ctrl.Result{RequeueAfter: v1alpha1.DefaultProgressingRequeue}, nil
}

func (helper *StatusHelper) MarkSucceeded(ctx context.Context, instance *v1alpha1.ApplicationGroup) error {
	// Set the last successful annotation for rollback scenarios
	instanceCopy := instance.DeepCopy()
	instanceCopy.SetLastSuccessful()
	if err := helper.Patch(ctx, instanceCopy, helper.PatchFrom); err != nil {
		helper.V(1).Error(err, "failed to patch the application group annotations")
		return err
	}

	// Set the status conditions into a succeeding state
	instance.ReadySucceeded()
	helper.Recorder.Event(instance, "Normal", "ReconcileSuccess", fmt.Sprintf("Successfully reconciled ApplicationGroup %s", instance.Name))
	return nil
}

// MarkProgressing resets the conditions of the ApplicationGroup to
// metav1.Condition of type meta.ReadyCondition with status 'Unknown' and
// meta.StartingReason reason and message.
func (helper *StatusHelper) MarkProgressing(ctx context.Context, instance *v1alpha1.ApplicationGroup) error {
	instance.Status.Conditions = []metav1.Condition{}
	meta.SetResourceCondition(instance, meta.ReadyCondition, metav1.ConditionUnknown, meta.ProgressingReason, "workflow is reconciling...")

	return helper.PatchStatus(ctx, instance)
}

// MarkTerminating sets the meta.ReadyCondition to 'False', with the given
// meta.Terminating reason and message
func (helper *StatusHelper) MarkTerminating(instance *v1alpha1.ApplicationGroup) {
	meta.SetResourceCondition(instance, meta.ReadyCondition, metav1.ConditionFalse, meta.TerminatingReason, "application group is terminating...")
}

// MarkFailed sets the meta.ReadyCondition to 'False', with a failed reason
func (helper *StatusHelper) MarkFailed(instance *v1alpha1.ApplicationGroup, err error) {
	helper.Recorder.Event(instance, "Warning", "ReconcileError", err.Error())
	instance.WorkflowFailed(err.Error())
}

// MarkChartPullFailed sets the meta.ReadyCondition to 'False', with a chart pull failed reason
func (helper *StatusHelper) MarkChartPullFailed(instance *v1alpha1.ApplicationGroup, err error) {
	helper.Recorder.Event(instance, "Warning", "ReconcileError", err.Error())
	instance.ChartPullFailed(err.Error())
}

// MarkWorkflowTemplateGenerationFailed sets the meta.ReadyCondition to 'False', with a workflow template generation failed reason
func (helper *StatusHelper) MarkWorkflowTemplateGenerationFailed(instance *v1alpha1.ApplicationGroup, err error) {
	helper.Recorder.Event(instance, "Warning", "ReconcileError", err.Error())
	instance.WorkflowTemplateGenerationFailed(err.Error())
}

func initAppStatus(appGroup *v1alpha1.ApplicationGroup) {
	// Initialize the Status fields if not already setup
	appGroup.Status.Applications = make([]v1alpha1.ApplicationStatus, 0, len(appGroup.Spec.Applications))
	for _, app := range appGroup.Spec.Applications {
		status := v1alpha1.ApplicationStatus{
			Name:        app.Name,
			ChartStatus: v1alpha1.ChartStatus{Version: app.Spec.Chart.Version},
			Subcharts:   make(map[string]v1alpha1.ChartStatus),
		}
		appGroup.Status.Applications = append(appGroup.Status.Applications, status)
	}
}

// marshallChartStatus lists all of the HelmRelease objects that were deployed and assigns
// their status to the appropriate maps corresponding to their chart of subchart.
// These statuses are used to update the application status above
func (helper *StatusHelper) marshallChartStatus(ctx context.Context, appGroup *v1alpha1.ApplicationGroup) (
	chartConditionMap map[string][]metav1.Condition,
	subChartConditionMap map[string]map[string][]metav1.Condition,
	err error) {
	listOption := client.MatchingLabels{
		v1alpha1.OwnershipLabel: appGroup.Name,
		v1alpha1.HeritageLabel:  v1alpha1.HeritageValue,
	}

	// Init the mappings
	chartConditionMap = make(map[string][]metav1.Condition)
	subChartConditionMap = make(map[string]map[string][]metav1.Condition)

	// Lookup all associated HelmReleases for status as well since the Workflow will not always reflect the status of the HelmRelease
	// Lookup Workflow by ownership and heritage labels
	helmReleases := fluxhelmv2beta1.HelmReleaseList{}
	err = helper.List(ctx, &helmReleases, listOption)
	if err != nil {
		helper.Error(err, "failed to find generated HelmRelease instance")
		return nil, nil, err
	}

	for _, hr := range helmReleases.Items {
		parent := hr.Name
		if v, ok := hr.GetAnnotations()["orkestra/parent-chart"]; ok {
			// Use the parent charts name
			parent = v
		}

		// Add the associated conditions for that helm chart to the helm chart condition
		// If the helm chart is a subchart, then add that to the subchart condition
		if parent == hr.Name {
			chartConditionMap[parent] = append(chartConditionMap[parent], hr.Status.Conditions...)
		} else {
			if _, ok := subChartConditionMap[parent]; !ok {
				subChartConditionMap[parent] = make(map[string][]metav1.Condition)
			}
			subChartConditionMap[parent][hr.Spec.ReleaseName] = append(subChartConditionMap[parent][hr.Spec.ReleaseName], hr.Status.Conditions...)
		}
	}
	return chartConditionMap, subChartConditionMap, nil
}

func getAppStatus(appGroup *v1alpha1.ApplicationGroup, chartConditionMap map[string][]metav1.Condition,
	subChartConditionMap map[string]map[string][]metav1.Condition) []v1alpha1.ApplicationStatus {
	// Update each application status using the HelmRelease status
	var v []v1alpha1.ApplicationStatus
	for _, app := range appGroup.Status.Applications {
		app.ChartStatus.Conditions = chartConditionMap[app.Name]
		for subchartName, subchartStatus := range app.Subcharts {
			subchartStatus.Conditions = subChartConditionMap[app.Name][subchartName]
			app.Subcharts[subchartName] = subchartStatus
		}
		v = append(v, app)
	}
	return v
}

func (helper *StatusHelper) PatchStatus(ctx context.Context, instance *v1alpha1.ApplicationGroup) error {
	if err := helper.Status().Patch(ctx, instance, helper.PatchFrom); err != nil {
		helper.V(1).Error(err, "failed to patch the application group status")
		return err
	}
	return nil
}
