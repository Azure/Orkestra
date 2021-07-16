package helpers

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	"github.com/Azure/Orkestra/pkg/registry"
	"github.com/Azure/Orkestra/pkg/utils"
	"github.com/Azure/Orkestra/pkg/workflow"
	"github.com/go-logr/logr"
	"github.com/jinzhu/copier"
	"helm.sh/helm/v3/pkg/chart"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	dummyConfigmapYAMLSpec = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-dummy
  namespace: {{ .Release.Namespace }} 
data:
  name: {{ .Chart.Name }}
  version : {{ .Chart.Version }}
`
	dummyConfigmapYAMLName = "templates/dummy-configmap.yaml"
)

type ReconcileHelper struct {
	client.Client
	logr.Logger
	Instance              *v1alpha1.ApplicationGroup
	WorkflowClientBuilder *workflow.Builder
	RegistryClient        *registry.Client
	StatusHelper          *StatusHelper

	RegistryOptions RegistryClientOptions
}

type RegistryClientOptions struct {
	StagingRepoName         string
	TargetDir               string
	CleanupDownloadedCharts bool
	stagingDirecotry        string
}

func (helper *ReconcileHelper) CreateOrUpdate(ctx context.Context) error {
	helper.Logger = helper.Logger.WithValues(v1alpha1.AppGroupNameKey, helper.Instance.Name)
	helper.V(3).Info("Reconciling ApplicationGroup object")

	if err := helper.reconcileApplications(); err != nil {
		helper.StatusHelper.MarkChartPullFailed(helper.Instance, err)
		return fmt.Errorf("failed to reconcile the applications with: %w", err)
	}
	// Generate the Workflow object to submit to Argo
	forwardClient := helper.WorkflowClientBuilder.Forward(helper.Instance).Build()
	if err := workflow.Run(ctx, forwardClient); err != nil {
		helper.StatusHelper.MarkWorkflowTemplateGenerationFailed(helper.Instance, err)
		return fmt.Errorf("failed to run forward workflow with: %w", err)
	}
	helper.Instance.Status.ObservedGeneration = helper.Instance.Generation
	return nil
}

func (helper *ReconcileHelper) Rollback(ctx context.Context, patch client.Patch) (ctrl.Result, error) {
	helper.Info("Rolling back to last successful application group spec")
	rollbackClient := helper.WorkflowClientBuilder.Rollback(helper.Instance).Build()

	// Re-running the workflow will not re-generate it since we check if we have already started it
	if err := workflow.Run(ctx, rollbackClient); err != nil {
		helper.Error(err, "failed to create the workflow for rollback")
		return ctrl.Result{}, err
	}
	if isSucceeded, err := workflow.IsSucceeded(ctx, rollbackClient); err != nil {
		helper.Error(err, "failed to validate if the workflow is succeeded")
		return ctrl.Result{}, err
	} else if isSucceeded {
		return ctrl.Result{}, nil
	}
	return reconcile.Result{RequeueAfter: v1alpha1.DefaultProgressingRequeue}, nil
}

func (helper *ReconcileHelper) Reverse(ctx context.Context) (ctrl.Result, error) {
	reverseClient := helper.WorkflowClientBuilder.Reverse(helper.Instance).Build()
	forwardClient := helper.WorkflowClientBuilder.Forward(helper.Instance).Build()
	helper.Info("Reversing the workflow")

	// Re-running the workflow will not re-generate it since we check if we have already started it
	if err := workflow.Run(ctx, reverseClient); errors.Is(err, meta.ErrForwardWorkflowNotFound) {
		// Forward workflow wasn't found so we just return
		return ctrl.Result{}, nil
	} else if err != nil {
		helper.Error(err, "failed to generate reverse workflow")
		// if generation of reverse workflow failed, delete the forward workflow and return
		if err := workflow.DeleteWorkflow(ctx, forwardClient); err != nil {
			helper.Error(err, "failed to delete workflow CRO")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if isSucceeded, err := workflow.IsSucceeded(ctx, reverseClient); err != nil {
		helper.Error(err, "failed to validate if the workflow is succeeded")
		return ctrl.Result{}, err
	} else if isSucceeded {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{Requeue: true, RequeueAfter: v1alpha1.DefaultProgressingRequeue}, nil
}

func (helper *ReconcileHelper) reconcileApplications() error {
	// Init the application status every time we re-reconcile the applications
	initAppStatus(helper.Instance)

	// Pull and conditionally stage application & dependency charts
	for i, application := range helper.Instance.Spec.Applications {
		ll := helper.WithValues("application", application.Name)
		ll.V(3).Info("performing chart actions")

		repoCfg, err := registry.GetHelmRepoConfig(&application, helper.Client)
		if err != nil {
			return fmt.Errorf("failed to get repo configuration for repo at URL %s: %w", application.Spec.Chart.URL, err)
		}

		if err := helper.RegistryClient.AddRepo(repoCfg); err != nil {
			return fmt.Errorf("failed to add helm repo at URL %s: %w", application.Spec.Chart.URL, err)
		}

		name := application.Spec.Chart.Name
		version := application.Spec.Chart.Version
		repoKey := application.Name

		fpath, appCh, err := helper.RegistryClient.PullChart(ll, repoKey, name, version)
		defer func() {
			if helper.RegistryOptions.CleanupDownloadedCharts {
				os.Remove(fpath)
			}
		}()
		if err != nil || appCh == nil {
			return fmt.Errorf("failed to pull application chart %s/%s:%s: %w", repoKey, name, version, err)
		}

		var mustStageSubcharts bool

		if application.Spec.Subcharts != nil && len(application.Spec.Subcharts) > 0 && appCh.Dependencies() != nil {
			mustStageSubcharts = true
		}

		if mustStageSubcharts {
			// take account of all embedded subcharts found in the application chart
			embeddedSubcharts := make(map[string]bool)
			for _, d := range appCh.Dependencies() {
				embeddedSubcharts[d.Name()] = true
			}

			// Remove all explicit subchart entries from tracking map
			for _, d := range helper.Instance.Spec.Applications[i].Spec.Subcharts {
				delete(embeddedSubcharts, d.Name)
			}

			// Use the remaining set of dependencies that are not explicitly declared and
			// add them to the groups application spec's subcharts list
			for name := range embeddedSubcharts {
				helper.Instance.Spec.Applications[i].Spec.Subcharts = append(helper.Instance.Spec.Applications[i].Spec.Subcharts, v1alpha1.DAG{Name: name})
			}

			stagingRepoName := helper.RegistryOptions.StagingRepoName
			// Package and push all application subcharts staging registry
			for _, sc := range appCh.Dependencies() {
				chartStatus := v1alpha1.ChartStatus{
					Version: sc.Metadata.Version,
					Staged:  false,
					Error:   "",
				}

				// copy the subchart
				scc := &chart.Chart{}
				_ = copier.Copy(scc, sc)

				if err := scc.Validate(); err != nil {
					err = fmt.Errorf("failed to validate application subchart for staging registry: %w", err)
					chartStatus.Error = err.Error()
					helper.Instance.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
					return err
				}

				// Copy over all non yaml files from parent chart templates to subchart templates
				for _, f := range appCh.Templates {
					if !utils.IsFileYaml(f.Name) {
						t := &chart.File{}
						_ = copier.Copy(t, f)
						t.Name = utils.AddAppChartNameToFile(t.Name, appCh.Name())
						scc.Templates = append(scc.Templates, t)
					}
				}

				scc.Metadata.Name = utils.GetSubchartName(appCh.Metadata.Name, scc.Metadata.Name)
				path, err := registry.SaveChartPackage(scc, helper.getStagingDirectory())
				if err != nil {
					err = fmt.Errorf("failed to save subchart package as tgz at location %s: %w", path, err)
					chartStatus.Error = err.Error()
					helper.Instance.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
					return err
				}

				err = helper.RegistryClient.PushChart(ll, stagingRepoName, path, scc)
				if err != nil {
					err = fmt.Errorf("failed to push application subchart to staging registry: %w", err)
					chartStatus.Error = err.Error()
					helper.Instance.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
					return err
				}

				chartStatus.Staged = true

				helper.Instance.Status.Applications[i].Subcharts[sc.Name()] = chartStatus
			}
		}

		// Unset dependencies by disabling them.
		// Using appCh.SetDependencies() does not cut it since some charts rely on subcharts for tpl helpers
		// provided in the charts directory.
		// IMPORTANT: This expects charts to follow best practices to allow enabling and disabling subcharts
		// See: https://helm.sh/docs/topics/charts/ #Chart Dependencies
		if mustStageSubcharts {
			for _, dep := range appCh.Metadata.Dependencies {
				// Disable subchart through metadata
				dep.Enabled = false
				// Precautionary - overwrite values with subcharts disabled
				appCh.Values[dep.Name] = map[string]interface{}{
					"enabled": false,
				}
			}
		}

		templateHasYAML, err := utils.TemplateContainsYaml(appCh)
		if err != nil {
			err = fmt.Errorf("chart templates directory yaml check failed: %w", err)
			helper.Instance.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		// If the parent chart doesnt contain any templates and all subcharts (if any) have been disabled we must create a dummy yaml to circumvent https://github.com/helm/helm/issues/4670
		if appCh.Templates == nil || len(appCh.Templates) == 0 || !templateHasYAML {
			if appCh.Templates == nil {
				appCh.Templates = make([]*chart.File, 0)
			}
			dummy := &chart.File{
				Name: dummyConfigmapYAMLName,
				Data: []byte(dummyConfigmapYAMLSpec),
			}
			appCh.Templates = append(appCh.Templates, dummy)
		}

		if err := appCh.Validate(); err != nil {
			err = fmt.Errorf("failed to validate application chart for staging registry: %w", err)
			helper.Instance.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		appCh.Metadata.Name = utils.ConvertToDNS1123(appCh.Metadata.Name)

		_, err = registry.SaveChartPackage(appCh, helper.getStagingDirectory())
		if err != nil {
			err = fmt.Errorf("failed to save modified app chart to filesystem: %w", err)
			helper.Instance.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		// Replace existing chart with modified chart
		chartPath := helper.getChartPath(application.Spec.Chart.Name, appCh.Metadata.Version)
		err = helper.RegistryClient.PushChart(ll, helper.RegistryOptions.StagingRepoName, chartPath, appCh)
		defer func() {
			if helper.RegistryOptions.CleanupDownloadedCharts {
				os.Remove(chartPath)
			}
		}()
		if err != nil {
			err = fmt.Errorf("failed to push modified application chart to staging registry: %w", err)
			helper.Instance.Status.Applications[i].ChartStatus.Error = err.Error()
			return err
		}

		helper.Instance.Status.Applications[i].ChartStatus.Staged = true
	}
	return nil
}

func (helper *ReconcileHelper) getStagingDirectory() string {
	return helper.RegistryOptions.TargetDir + "/" + helper.RegistryOptions.StagingRepoName
}

func (helper *ReconcileHelper) getChartPath(name, version string) string {
	return helper.getStagingDirectory() + "/" + utils.ConvertToDNS1123(name) + "-" + version + ".tgz"
}
