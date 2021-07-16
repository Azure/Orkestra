package workflow

import (
	"fmt"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxsourcev1beta1 "github.com/fluxcd/source-controller/api/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateTemplates(instance *v1alpha1.ApplicationGroup, options ClientOptions) (*v1alpha13.Template, []v1alpha13.Template, error) {
	if instance == nil {
		return nil, nil, fmt.Errorf("applicationGroup cannot be nil")
	}
	templates, err := generateAppDAGTemplates(instance, options.namespace, options.parallelism)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate application DAG templates : %w", err)
	}

	// Create the entry template from the app dag templates
	entryTemplate := &v1alpha13.Template{
		Name: EntrypointTemplateName,
		DAG: &v1alpha13.DAGTemplate{
			Tasks: make([]v1alpha13.DAGTask, len(instance.Spec.Applications)),
		},
		Parallelism: options.parallelism,
	}
	for i, tpl := range templates {
		entryTemplate.DAG.Tasks[i] = v1alpha13.DAGTask{
			Name:         utils.ConvertToDNS1123(tpl.Name),
			Template:     utils.ConvertToDNS1123(tpl.Name),
			Dependencies: utils.ConvertSliceToDNS1123(instance.Spec.Applications[i].Dependencies),
		}
	}
	return entryTemplate, templates, nil
}

func generateAppDAGTemplates(appGroup *v1alpha1.ApplicationGroup, namespace string, parallelism *int64) ([]v1alpha13.Template, error) {
	ts := make([]v1alpha13.Template, 0)

	for i, app := range appGroup.Spec.Applications {
		var hasSubcharts bool
		scStatus := appGroup.Status.Applications[i].Subcharts

		// Create Subchart DAG only when the application chart has dependencies
		if len(app.Spec.Subcharts) > 0 {
			hasSubcharts = true
			t := v1alpha13.Template{
				Name:        utils.ConvertToDNS1123(app.Name),
				Parallelism: parallelism,
			}

			t.DAG = &v1alpha13.DAGTemplate{}
			tasks, err := generateSubchartAndAppDAGTasks(appGroup.Name, namespace, &app, scStatus)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Application Template DAG tasks : %w", err)
			}

			t.DAG.Tasks = tasks

			ts = append(ts, t)
		}

		if !hasSubcharts {
			hr := helmReleaseBuilder(app.Spec.Release, namespace, app.Name, app.Spec.Chart.Name, app.Name, app.Spec.Chart.Version)
			hr.Spec.Interval = app.Spec.Release.Interval
			hr.Spec.Values = app.Spec.Release.Values
			hr.Labels = map[string]string{
				v1alpha1.ChartLabel:     app.Name,
				v1alpha1.OwnershipLabel: appGroup.Name,
				v1alpha1.HeritageLabel:  v1alpha1.HeritageValue,
			}
			hrStr := utils.HrToB64AnyStringPtr(hr)

			tApp := v1alpha13.Template{
				Name:        utils.ConvertToDNS1123(app.Name),
				Parallelism: parallelism,
				DAG: &v1alpha13.DAGTemplate{
					Tasks: []v1alpha13.DAGTask{
						appDAGTaskBuilder(app.Name, getTimeout(app.Spec.Release.Timeout), hrStr),
					},
				},
			}
			ts = append(ts, tApp)
		}
	}
	return ts, nil
}

func generateSubchartAndAppDAGTasks(appGroupName, namespace string, app *v1alpha1.Application, subchartsStatus map[string]v1alpha1.ChartStatus) ([]v1alpha13.DAGTask, error) {
	// XXX (nitishm) Should this be set to nil if no subcharts are found??
	tasks := make([]v1alpha13.DAGTask, 0, len(app.Spec.Subcharts)+1)

	for _, sc := range app.Spec.Subcharts {
		subchartName := sc.Name
		subchartVersion := subchartsStatus[subchartName].Version

		hr, err := generateSubchartHelmRelease(app.Spec.Release, namespace, app.Spec.Chart.Name, subchartName, subchartVersion)
		if err != nil {
			return nil, err
		}
		hr.Annotations = map[string]string{
			v1alpha1.ParentChartAnnotation: app.Name,
		}
		hr.Labels = map[string]string{
			v1alpha1.ChartLabel:     app.Name,
			v1alpha1.OwnershipLabel: appGroupName,
			v1alpha1.HeritageLabel:  v1alpha1.HeritageValue,
		}
		hrStr := utils.HrToB64AnyStringPtr(hr)

		task := appDAGTaskBuilder(subchartName, getTimeout(app.Spec.Release.Timeout), hrStr)
		task.Dependencies = utils.ConvertSliceToDNS1123(sc.Dependencies)
		tasks = append(tasks, task)
	}

	hr := helmReleaseBuilder(app.Spec.Release, namespace, app.Name, app.Spec.Chart.Name, app.Name, app.Spec.Chart.Version)
	hr.Spec.Interval = app.Spec.Release.Interval
	hr.Spec.Values = app.Spec.Release.Values
	hr.Labels = map[string]string{
		v1alpha1.ChartLabel:     app.Name,
		v1alpha1.OwnershipLabel: appGroupName,
		v1alpha1.HeritageLabel:  v1alpha1.HeritageValue,
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

	hrStr := utils.HrToB64AnyStringPtr(hr)
	task := appDAGTaskBuilder(app.Name, getTimeout(app.Spec.Release.Timeout), hrStr)
	task.Dependencies = func() (out []string) {
		for _, t := range tasks {
			out = append(out, utils.ConvertToDNS1123(t.Name))
		}
		return out
	}()
	tasks = append(tasks, task)

	return tasks, nil
}

func appDAGTaskBuilder(name string, timeout, hrStr *v1alpha13.AnyString) v1alpha13.DAGTask {
	task := v1alpha13.DAGTask{
		Name:     utils.ConvertToDNS1123(name),
		Template: HelmReleaseExecutorName,
		Arguments: v1alpha13.Arguments{
			Parameters: []v1alpha13.Parameter{
				{
					Name:  HelmReleaseArg,
					Value: hrStr,
				},
				{
					Name:  TimeoutArg,
					Value: timeout,
				},
			},
		},
	}
	return task
}

func generateSubchartHelmRelease(r *v1alpha1.Release, namespace, appChartName, subchartName, version string) (*fluxhelmv2beta1.HelmRelease, error) {
	chName := utils.GetSubchartName(appChartName, subchartName)
	hr := helmReleaseBuilder(r, namespace, chName, chName, subchartName, version)

	var releaseValues map[string]interface{}
	if r != nil {
		releaseValues = r.GetValues()
	}
	val, err := subchartValues(subchartName, releaseValues)
	if err != nil {
		return nil, err
	}
	hr.Spec.Values = val
	return hr, nil
}

func helmReleaseBuilder(r *v1alpha1.Release, namespace, objMetaName, chName, releaseName, version string) *fluxhelmv2beta1.HelmRelease {
	if r == nil {
		r = &v1alpha1.Release{}
	}
	hr := &fluxhelmv2beta1.HelmRelease{
		TypeMeta: v1.TypeMeta{
			Kind:       fluxhelmv2beta1.HelmReleaseKind,
			APIVersion: fluxhelmv2beta1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      utils.ConvertToDNS1123(objMetaName),
			Namespace: r.TargetNamespace,
		},
		Spec: fluxhelmv2beta1.HelmReleaseSpec{
			Chart: fluxhelmv2beta1.HelmChartTemplate{
				Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
					Chart:   utils.ConvertToDNS1123(chName),
					Version: version,
					SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
						Kind:      fluxsourcev1beta1.HelmRepositoryKind,
						Name:      ChartMuseumName,
						Namespace: namespace,
					},
				},
			},
			ReleaseName:     utils.ConvertToDNS1123(releaseName),
			TargetNamespace: r.TargetNamespace,
			Timeout:         r.Timeout,
			Install:         r.Install,
			Upgrade:         r.Upgrade,
			Rollback:        r.Rollback,
			Uninstall:       r.Uninstall,
		},
	}
	return hr
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
