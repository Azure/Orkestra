package workflow

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxsourcev1beta1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (engine ForwardEngine) GetLogger() logr.Logger {
	return engine.Logger
}

func (engine ForwardEngine) Generate(ctx context.Context) error {
	if engine.appGroup == nil {
		engine.Error(nil, "ApplicationGroup object cannot be nil")
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	engine.workflow = initWorkflowObject(engine.parallelism)

	// Set name and namespace based on the input application group
	engine.workflow.Name = engine.appGroup.Name
	engine.workflow.Namespace = workflowNamespace()

	entryTemplate, templates, err := engine.generateTemplates(ctx)
	if err != nil {
		engine.Error(err, "failed to generate workflow")
		return fmt.Errorf("failed to generate argo workflow : %w", err)
	}

	updateWorkflowTemplates(engine.workflow, templates...)

	err = updateAppGroupDAG(engine.appGroup, entryTemplate, templates)
	if err != nil {
		return fmt.Errorf("failed to generate Application Group DAG : %w", err)
	}
	updateWorkflowTemplates(engine.workflow, *entryTemplate)

	// TODO: Add the executor template
	// This should eventually be configurable
	updateWorkflowTemplates(engine.workflow, defaultExecutor(HelmReleaseExecutorName, Install))

	return nil
}

func (engine ForwardEngine) Submit(ctx context.Context) error {
	if engine.workflow == nil {
		engine.Error(nil, "workflow object cannot be nil")
		return fmt.Errorf("workflow object cannot be nil")
	}

	if engine.appGroup == nil {
		engine.Error(nil, "applicationGroup object cannot be nil")
		return fmt.Errorf("applicationGroup object cannot be nil")
	}

	namespaces := []string{}
	// Add namespaces we need to create while removing duplicates
	for _, app := range engine.appGroup.Spec.Applications {
		found := false
		for _, namespace := range namespaces {
			if app.Spec.Release.TargetNamespace == namespace {
				found = true
				break
			}
		}
		if !found {
			namespaces = append(namespaces, app.Spec.Release.TargetNamespace)
		}
	}

	// Create any of the target namespaces
	for _, namespace := range namespaces {
		ns := &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: namespace,
			},
		}
		if err := controllerutil.SetControllerReference(engine.appGroup, ns, engine.Scheme()); err != nil {
			return fmt.Errorf("failed to set OwnerReference for Namespace %s : %w", ns.Name, err)
		}
		if err := engine.Create(ctx, ns); !errors.IsAlreadyExists(err) && err != nil {
			return fmt.Errorf("failed to CREATE namespace %s object : %w", ns.Name, err)
		}
	}

	// Create the Workflow
	engine.workflow.Labels[OwnershipLabel] = engine.appGroup.Name
	if err := controllerutil.SetControllerReference(engine.appGroup, engine.workflow, engine.Scheme()); err != nil {
		engine.Error(err, "unable to set ApplicationGroup as owner of Argo Workflow object")
		return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
	}
	if err := engine.Create(ctx, engine.workflow); !errors.IsAlreadyExists(err) && err != nil {
		engine.Error(err, "failed to CREATE argo workflow object")
		return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
	} else if errors.IsAlreadyExists(err) {
		// If the workflow needs an update, delete the previous workflow and apply the new one
		// Argo Workflow does not rerun the workflow on UPDATE, so intead we cleanup and reapply
		if err := engine.Delete(ctx, engine.workflow); err != nil {
			engine.Error(err, "failed to DELETE argo workflow object")
			return fmt.Errorf("failed to DELETE argo workflow object : %w", err)
		}
		if err := controllerutil.SetControllerReference(engine.appGroup, engine.workflow, engine.Scheme()); err != nil {
			engine.Error(err, "unable to set ApplicationGroup as owner of Argo Workflow object")
			return fmt.Errorf("unable to set ApplicationGroup as owner of Argo Workflow: %w", err)
		}
		// If the argo Workflow object is NotFound and not AlreadyExists on the cluster
		// create a new object and submit it to the cluster
		if err := engine.Create(ctx, engine.workflow); err != nil {
			engine.Error(err, "failed to CREATE argo workflow object")
			return fmt.Errorf("failed to CREATE argo workflow object : %w", err)
		}
	}
	return nil
}

func (engine ForwardEngine) generateTemplates(ctx context.Context) (*v1alpha12.Template, []v1alpha12.Template, error) {
	if engine.appGroup == nil {
		return nil, nil, fmt.Errorf("applicationGroup cannot be nil")
	}

	entryTemplate := &v1alpha12.Template{
		Name: EntrypointTemplateName,
		DAG: &v1alpha12.DAGTemplate{
			Tasks: make([]v1alpha12.DAGTask, len(engine.appGroup.Spec.Applications)),
			// TBD (nitishm): Do we need to failfast?
			// FailFast: true
		},
		Parallelism: engine.parallelism,
	}

	templates, err := engine.generateAppDAGTemplates()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate application DAG templates : %w", err)
	}
	return entryTemplate, templates, nil
}

func (engine ForwardEngine) generateAppDAGTemplates() ([]v1alpha12.Template, error) {
	ts := make([]v1alpha12.Template, 0)

	for i, app := range engine.appGroup.Spec.Applications {
		var hasSubcharts bool
		scStatus := engine.appGroup.Status.Applications[i].Subcharts

		// Create Subchart DAG only when the application chart has dependencies
		if len(app.Spec.Subcharts) > 0 {
			hasSubcharts = true
			t := v1alpha12.Template{
				Name:        utils.ConvertToDNS1123(app.Name),
				Parallelism: engine.parallelism,
			}

			t.DAG = &v1alpha12.DAGTemplate{}
			tasks, err := engine.generateSubchartAndAppDAGTasks(&app, scStatus)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Application Template DAG tasks : %w", err)
			}

			t.DAG.Tasks = tasks

			ts = append(ts, t)
		}

		if !hasSubcharts {
			hr := fluxhelmv2beta1.HelmRelease{
				TypeMeta: v1.TypeMeta{
					Kind:       fluxhelmv2beta1.HelmReleaseKind,
					APIVersion: fluxhelmv2beta1.GroupVersion.String(),
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      utils.ConvertToDNS1123(app.Name),
					Namespace: app.Spec.Release.TargetNamespace,
				},
				Spec: fluxhelmv2beta1.HelmReleaseSpec{
					Chart: fluxhelmv2beta1.HelmChartTemplate{
						Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
							Chart:   utils.ConvertToDNS1123(app.Spec.Chart.Name),
							Version: app.Spec.Chart.Version,
							SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
								Kind:      fluxsourcev1beta1.HelmRepositoryKind,
								Name:      ChartMuseumName,
								Namespace: workflowNamespace(),
							},
						},
					},
					Interval:        app.Spec.Release.Interval,
					ReleaseName:     utils.ConvertToDNS1123(app.Name),
					TargetNamespace: app.Spec.Release.TargetNamespace,
					Timeout:         app.Spec.Release.Timeout,
					Values:          app.Spec.Release.Values,
					Install:         app.Spec.Release.Install,
					Upgrade:         app.Spec.Release.Upgrade,
					Rollback:        app.Spec.Release.Rollback,
					Uninstall:       app.Spec.Release.Uninstall,
				},
			}
			hr.Labels = map[string]string{
				ChartLabelKey:  app.Name,
				OwnershipLabel: engine.appGroup.Name,
				HeritageLabel:  Project,
			}

			tApp := v1alpha12.Template{
				Name:        utils.ConvertToDNS1123(app.Name),
				Parallelism: engine.parallelism,
				DAG: &v1alpha12.DAGTemplate{
					Tasks: []v1alpha12.DAGTask{
						{
							Name:     utils.ConvertToDNS1123(app.Name),
							Template: HelmReleaseExecutorName,
							Arguments: v1alpha12.Arguments{
								Parameters: []v1alpha12.Parameter{
									{
										Name:  HelmReleaseArg,
										Value: utils.ToStrPtr(base64.StdEncoding.EncodeToString([]byte(utils.HrToYaml(hr)))),
									},
									{
										Name:  TimeoutArg,
										Value: getTimeout(app.Spec.Release.Timeout),
									},
								},
							},
						},
					},
				},
			}

			ts = append(ts, tApp)
		}
	}
	return ts, nil
}

func (engine ForwardEngine) generateSubchartAndAppDAGTasks(app *v1alpha1.Application, subchartsStatus map[string]v1alpha1.ChartStatus) ([]v1alpha12.DAGTask, error) {
	if engine.stagingRepo == "" {
		return nil, fmt.Errorf("repo arg must be a valid non-empty string")
	}

	// XXX (nitishm) Should this be set to nil if no subcharts are found??
	tasks := make([]v1alpha12.DAGTask, 0, len(app.Spec.Subcharts)+1)

	for _, sc := range app.Spec.Subcharts {
		subchartName := sc.Name
		subchartVersion := subchartsStatus[subchartName].Version

		hr, err := generateSubchartHelmRelease(*app, subchartName, subchartVersion)
		if err != nil {
			return nil, err
		}
		hr.Annotations = map[string]string{
			v1alpha1.ParentChartAnnotation: app.Name,
		}
		hr.Labels = map[string]string{
			ChartLabelKey:  app.Name,
			OwnershipLabel: engine.appGroup.Name,
			HeritageLabel:  Project,
		}

		task := v1alpha12.DAGTask{
			Name:     utils.ConvertToDNS1123(subchartName),
			Template: HelmReleaseExecutorName,
			Arguments: v1alpha12.Arguments{
				Parameters: []v1alpha12.Parameter{
					{
						Name:  HelmReleaseArg,
						Value: utils.ToStrPtr(base64.StdEncoding.EncodeToString([]byte(utils.HrToYaml(*hr)))),
					},
					{
						Name:  TimeoutArg,
						Value: getTimeout(app.Spec.Release.Timeout),
					},
				},
			},
			Dependencies: utils.ConvertSliceToDNS1123(sc.Dependencies),
		}

		tasks = append(tasks, task)
	}

	hr := fluxhelmv2beta1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       fluxhelmv2beta1.HelmReleaseKind,
			APIVersion: fluxhelmv2beta1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.ConvertToDNS1123(app.Name),
			Namespace: app.Spec.Release.TargetNamespace,
		},
		Spec: fluxhelmv2beta1.HelmReleaseSpec{
			Chart: fluxhelmv2beta1.HelmChartTemplate{
				Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
					Chart:   utils.ConvertToDNS1123(app.Spec.Chart.Name),
					Version: app.Spec.Chart.Version,
					SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
						Kind:      fluxsourcev1beta1.HelmRepositoryKind,
						Name:      ChartMuseumName,
						Namespace: workflowNamespace(),
					},
				},
			},
			Interval:        app.Spec.Release.Interval,
			ReleaseName:     utils.ConvertToDNS1123(app.Name),
			TargetNamespace: app.Spec.Release.TargetNamespace,
			Timeout:         app.Spec.Release.Timeout,
			Values:          app.Spec.Release.Values,
			Install:         app.Spec.Release.Install,
			Upgrade:         app.Spec.Release.Upgrade,
			Rollback:        app.Spec.Release.Rollback,
			Uninstall:       app.Spec.Release.Uninstall,
		},
	}
	hr.Labels = map[string]string{
		ChartLabelKey:  app.Name,
		OwnershipLabel: engine.appGroup.Name,
		HeritageLabel:  Project,
	}

	// Force disable all subchart for the staged application chart
	// to prevent duplication and possible collision of deployed resources
	// Since the subchart should have been deployed in a prior DAG step,
	// we must not redeploy it along with the parent application chart.

	values := app.GetValues()
	for _, d := range app.Spec.Subcharts {
		values[d.Name] = map[string]interface{}{
			"enabled": false,
		}
	}
	if err := app.SetValues(values); err != nil {
		return nil, err
	}

	task := v1alpha12.DAGTask{
		Name:     utils.ConvertToDNS1123(app.Name),
		Template: HelmReleaseExecutorName,
		Arguments: v1alpha12.Arguments{
			Parameters: []v1alpha12.Parameter{
				{
					Name:  HelmReleaseArg,
					Value: utils.ToStrPtr(base64.StdEncoding.EncodeToString([]byte(utils.HrToYaml(hr)))),
				},
				{
					Name:  TimeoutArg,
					Value: getTimeout(app.Spec.Release.Timeout),
				},
			},
		},
		Dependencies: func() (out []string) {
			for _, t := range tasks {
				out = append(out, utils.ConvertToDNS1123(t.Name))
			}
			return out
		}(),
	}
	tasks = append(tasks, task)

	return tasks, nil
}

func generateSubchartHelmRelease(a v1alpha1.Application, subchartName, version string) (*fluxhelmv2beta1.HelmRelease, error) {
	hr := &fluxhelmv2beta1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       fluxhelmv2beta1.HelmReleaseKind,
			APIVersion: fluxhelmv2beta1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.ConvertToDNS1123(utils.ToInitials(a.Name) + "-" + subchartName),
			Namespace: a.Spec.Release.TargetNamespace,
		},
		Spec: fluxhelmv2beta1.HelmReleaseSpec{
			Chart: fluxhelmv2beta1.HelmChartTemplate{
				Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
					Chart:   utils.ConvertToDNS1123(utils.ToInitials(a.Name) + "-" + subchartName),
					Version: version,
					SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
						Kind:      fluxsourcev1beta1.HelmRepositoryKind,
						Name:      ChartMuseumName,
						Namespace: workflowNamespace(),
					},
				},
			},
			ReleaseName:     utils.ConvertToDNS1123(subchartName),
			TargetNamespace: a.Spec.Release.TargetNamespace,
			Timeout:         a.Spec.Release.Timeout,
			Install:         a.Spec.Release.Install,
			Upgrade:         a.Spec.Release.Upgrade,
			Rollback:        a.Spec.Release.Rollback,
			Uninstall:       a.Spec.Release.Uninstall,
		},
	}

	val, err := subchartValues(subchartName, a.GetValues())
	if err != nil {
		return nil, err
	}
	hr.Spec.Values = val
	return hr, nil
}

func subchartValues(sc string, values map[string]interface{}) (*apiextensionsv1.JSON, error) {
	data := make(map[string]interface{})

	if scVals, ok := values[sc]; ok {
		if vv, ok := scVals.(map[string]interface{}); ok {
			for k, val := range vv {
				data[k] = val
			}
		}
	}

	if gVals, ok := values[ValuesKeyGlobal]; ok {
		if vv, ok := gVals.(map[string]interface{}); ok {
			data[ValuesKeyGlobal] = vv
		}
	}

	return v1alpha1.GetJSON(data)
}

func getTaskNamesFromHelmReleases(bucket []fluxhelmv2beta1.HelmRelease) []string {
	out := []string{}
	for _, hr := range bucket {
		out = append(out, utils.ConvertToDNS1123(hr.GetReleaseName()+"-"+(hr.Namespace)))
	}
	return out
}

func updateAppGroupDAG(g *v1alpha1.ApplicationGroup, entry *v1alpha12.Template, tpls []v1alpha12.Template) error {
	if entry == nil {
		return fmt.Errorf("entry template cannot be nil")
	}

	entry.DAG = &v1alpha12.DAGTemplate{
		Tasks: make([]v1alpha12.DAGTask, len(tpls), len(tpls)),
	}

	for i, tpl := range tpls {
		entry.DAG.Tasks[i] = v1alpha12.DAGTask{
			Name:         utils.ConvertToDNS1123(tpl.Name),
			Template:     utils.ConvertToDNS1123(tpl.Name),
			Dependencies: utils.ConvertSliceToDNS1123(g.Spec.Applications[i].Dependencies),
		}
	}

	return nil
}