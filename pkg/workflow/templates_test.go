package workflow

import (
	"encoding/json"
	"testing"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/google/go-cmp/cmp"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_generateTemplates(t *testing.T) {
	type args struct {
		instance *v1alpha1.ApplicationGroup
		options  ClientOptions
	}
	var p int64 = 0
	tests := []struct {
		name    string
		args    args
		want1   *v1alpha13.Template
		want2   []v1alpha13.Template
		wantErr bool
	}{
		{
			name: "testing nil application",
			args: args{
				instance: nil,
				options: ClientOptions{
					parallelism: &p,
					stagingRepo: "http://stagingrepo",
					namespace:   "testorkestra",
				},
			},
			want1:   nil,
			want2:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1, got2, err := generateTemplates(tt.args.instance, tt.args.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateTemplates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got1, tt.want1) {
				t.Errorf("generateTemplates() = %v", cmp.Diff(got1, tt.want1))
			}
			if !cmp.Equal(got2, tt.want2) {
				t.Errorf("generateTemplates() = %v", cmp.Diff(got2, tt.want2))
			}
		})
	}
}

func Test_generateAppDAGTemplates(t *testing.T) {
	type args struct {
		appGroup    *v1alpha1.ApplicationGroup
		namespace   string
		parallelism *int64
	}
	relValues := map[string]interface{}{
		"global":     map[string]string{"keyG": "valueG"},
		"subchart-1": map[string]string{"sc1-key": "sc1-value"},
		"subchart-2": map[string]string{"sc2-key": "sc2-value"},
		"subchart-3": map[string]string{"sc3-key": "sc3-value"},
	}
	bytesRelValues, _ := json.Marshal(relValues)
	var p int64 = 0

	tests := []struct {
		name    string
		args    args
		want    []v1alpha13.Template
		wantErr bool
	}{
		{
			name: "testing singleApplicationWithSubchartDAG",
			args: args{
				appGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name: "application",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name:         "application",
									Dependencies: nil,
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										URL:     "http://stagingrepo",
										Name:    "appchart",
										Version: "0.1.6",
									},
									Release: &v1alpha1.Release{
										Values: &apiextensionsv1.JSON{
											Raw: bytesRelValues,
										},
									},
									Subcharts: []v1alpha1.DAG{
										{
											Name:         "subchart-3",
											Dependencies: []string{"subchart-2", "subchart-1"},
										},
										{
											Name:         "subchart-2",
											Dependencies: nil,
										},
										{
											Name:         "subchart-1",
											Dependencies: nil,
										},
									},
								},
							},
						},
					},
					Status: v1alpha1.ApplicationGroupStatus{
						Applications: []v1alpha1.ApplicationStatus{
							{
								Subcharts: map[string]v1alpha1.ChartStatus{
									"subchart-1": {
										Version: "1.0.0",
									},
									"subchart-2": {
										Version: "1.0.0",
									},
									"subchart-3": {
										Version: "1.0.0",
									},
								},
							},
						},
					},
				},
				namespace:   "testorkestra",
				parallelism: &p,
			},
			want: []v1alpha13.Template{
				{
					Name: "application",
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							{
								Name:     "subchart-3",
								Template: "helmrelease-executor",
								Arguments: v1alpha13.Arguments{
									Parameters: []v1alpha13.Parameter{
										{
											Name: "helmrelease",
											Value: utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "1ccd4cae89-subchart-3",
													Labels: map[string]string{
														ChartLabelKey:  "application",
														OwnershipLabel: "application",
														HeritageLabel:  "orkestra",
													},
													Annotations: map[string]string{
														v1alpha1.ParentChartAnnotation: "application",
													},
												},
												Spec: fluxhelmv2beta1.HelmReleaseSpec{
													Chart: fluxhelmv2beta1.HelmChartTemplate{
														Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
															Chart:   "1ccd4cae89-subchart-3",
															Version: "1.0.0",
															SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
																Kind:      "HelmRepository",
																Name:      "chartmuseum",
																Namespace: "testorkestra",
															},
														},
													},
													ReleaseName: "subchart-3",
													Values: &apiextensionsv1.JSON{
														Raw: []byte(`{"global":{"keyG":"valueG"},"sc3-key":"sc3-value"}`),
													},
												},
											}),
										},
										{
											Name:  "timeout",
											Value: utils.ToAnyStringPtr("5m"),
										},
									},
								},
								Dependencies: []string{"subchart-2", "subchart-1"},
							},
							{
								Name:     "subchart-2",
								Template: "helmrelease-executor",
								Arguments: v1alpha13.Arguments{
									Parameters: []v1alpha13.Parameter{
										{
											Name: "helmrelease",
											Value: utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "1ccd4cae89-subchart-2",
													Labels: map[string]string{
														ChartLabelKey:  "application",
														OwnershipLabel: "application",
														HeritageLabel:  "orkestra",
													},
													Annotations: map[string]string{
														v1alpha1.ParentChartAnnotation: "application",
													},
												},
												Spec: fluxhelmv2beta1.HelmReleaseSpec{
													Chart: fluxhelmv2beta1.HelmChartTemplate{
														Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
															Chart:   "1ccd4cae89-subchart-2",
															Version: "1.0.0",
															SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
																Kind:      "HelmRepository",
																Name:      "chartmuseum",
																Namespace: "testorkestra",
															},
														},
													},
													ReleaseName: "subchart-2",
													Values: &apiextensionsv1.JSON{
														Raw: []byte(`{"global":{"keyG":"valueG"},"sc2-key":"sc2-value"}`),
													},
												},
											}),
										},
										{
											Name:  "timeout",
											Value: utils.ToAnyStringPtr("5m"),
										},
									},
								},
								Dependencies: []string{},
							},
							{
								Name:     "subchart-1",
								Template: "helmrelease-executor",
								Arguments: v1alpha13.Arguments{
									Parameters: []v1alpha13.Parameter{
										{
											Name: "helmrelease",
											Value: utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "1ccd4cae89-subchart-1",
													Labels: map[string]string{
														ChartLabelKey:  "application",
														OwnershipLabel: "application",
														HeritageLabel:  "orkestra",
													},
													Annotations: map[string]string{
														v1alpha1.ParentChartAnnotation: "application",
													},
												},
												Spec: fluxhelmv2beta1.HelmReleaseSpec{
													Chart: fluxhelmv2beta1.HelmChartTemplate{
														Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
															Chart:   "1ccd4cae89-subchart-1",
															Version: "1.0.0",
															SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
																Kind:      "HelmRepository",
																Name:      "chartmuseum",
																Namespace: "testorkestra",
															},
														},
													},
													ReleaseName: "subchart-1",
													Values: &apiextensionsv1.JSON{
														Raw: []byte(`{"global":{"keyG":"valueG"},"sc1-key":"sc1-value"}`),
													},
												},
												Status: fluxhelmv2beta1.HelmReleaseStatus{},
											}),
										},
										{
											Name:  "timeout",
											Value: utils.ToAnyStringPtr("5m"),
										},
									},
								},
								Dependencies: []string{},
							},
							{
								Name:     "application",
								Template: "helmrelease-executor",
								Arguments: v1alpha13.Arguments{
									Parameters: []v1alpha13.Parameter{
										{
											Name: "helmrelease",
											Value: utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "application",
													Labels: map[string]string{
														ChartLabelKey:  "application",
														OwnershipLabel: "application",
														HeritageLabel:  "orkestra",
													},
												},
												Spec: fluxhelmv2beta1.HelmReleaseSpec{
													Chart: fluxhelmv2beta1.HelmChartTemplate{
														Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
															Chart:   "appchart",
															Version: "0.1.6",
															SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
																Kind:      "HelmRepository",
																Name:      "chartmuseum",
																Namespace: "testorkestra",
															},
														},
													},
													ReleaseName: "application",
													Values: &apiextensionsv1.JSON{
														Raw: []byte(`{"global":{"keyG":"valueG"},"subchart-1":{"enabled":false},"subchart-2":{"enabled":false},"subchart-3":{"enabled":false}}`),
													},
												},
											}),
										},
										{
											Name:  "timeout",
											Value: utils.ToAnyStringPtr("5m"),
										},
									},
								},
								Dependencies: []string{"subchart-3", "subchart-2", "subchart-1"},
							},
						},
					},
					Parallelism: &p,
				},
			},
			wantErr: false,
		},
		{
			name: "testing singleApplicationWithoutSubchartDAG",
			args: args{
				appGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name: "application",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name:         "application",
									Dependencies: nil,
								},
								Spec: v1alpha1.ApplicationSpec{
									Chart: &v1alpha1.ChartRef{
										URL:     "http://stagingrepo",
										Name:    "appchart",
										Version: "0.1.6",
									},
									Release: &v1alpha1.Release{
										Values: &apiextensionsv1.JSON{
											Raw: bytesRelValues,
										},
									},
								},
							},
						},
					},
					Status: v1alpha1.ApplicationGroupStatus{
						Applications: []v1alpha1.ApplicationStatus{
							{
								Subcharts: map[string]v1alpha1.ChartStatus{
									"subchart-1": {
										Version: "1.0.0",
									},
									"subchart-2": {
										Version: "1.0.0",
									},
									"subchart-3": {
										Version: "1.0.0",
									},
								},
							},
						},
					},
				},
				namespace:   "testorkestra",
				parallelism: &p,
			},
			want: []v1alpha13.Template{
				{
					Name: "application",
					DAG: &v1alpha13.DAGTemplate{
						Tasks: []v1alpha13.DAGTask{
							{
								Name:     "application",
								Template: "helmrelease-executor",
								Arguments: v1alpha13.Arguments{
									Parameters: []v1alpha13.Parameter{
										{
											Name: "helmrelease",
											Value: utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{
												TypeMeta: v1.TypeMeta{
													Kind:       "HelmRelease",
													APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
												},
												ObjectMeta: v1.ObjectMeta{
													Name: "application",
													Labels: map[string]string{
														ChartLabelKey:  "application",
														OwnershipLabel: "application",
														HeritageLabel:  "orkestra",
													},
												},
												Spec: fluxhelmv2beta1.HelmReleaseSpec{
													Chart: fluxhelmv2beta1.HelmChartTemplate{
														Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
															Chart:   "appchart",
															Version: "0.1.6",
															SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
																Kind:      "HelmRepository",
																Name:      "chartmuseum",
																Namespace: "testorkestra",
															},
														},
													},
													ReleaseName: "application",
													Values: &apiextensionsv1.JSON{
														Raw: []byte(`{"global":{"keyG":"valueG"},"subchart-1":{"sc1-key":"sc1-value"},"subchart-2":{"sc2-key":"sc2-value"},"subchart-3":{"sc3-key":"sc3-value"}}`),
													},
												},
											}),
										},
										{
											Name:  "timeout",
											Value: utils.ToAnyStringPtr("5m"),
										},
									},
								},
								Dependencies: nil,
							},
						},
					},
					Parallelism: &p,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateAppDAGTemplates(tt.args.appGroup, tt.args.namespace, tt.args.parallelism)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateAppDAGTemplates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("generateAppDAGTemplates() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_generateSubchartAndAppDAGTasks(t *testing.T) {
	type args struct {
		appGroupName    string
		namespace       string
		app             *v1alpha1.Application
		subchartsStatus map[string]v1alpha1.ChartStatus
	}
	relValues := map[string]interface{}{
		"global":     map[string]string{"keyG": "valueG"},
		"subchart-1": map[string]string{"sc1-key": "sc1-value"},
		"subchart-2": map[string]string{"sc2-key": "sc2-value"},
		"subchart-3": map[string]string{"sc3-key": "sc3-value"},
	}
	bytesRelValues, _ := json.Marshal(relValues)

	tests := []struct {
		name    string
		args    args
		want    []v1alpha13.DAGTask
		wantErr bool
	}{
		{
			name: "testing sequential",
			args: args{
				appGroupName: "application",
				namespace:    "testorkestra",
				app: &v1alpha1.Application{
					DAG: v1alpha1.DAG{
						Name:         "application",
						Dependencies: nil,
					},
					Spec: v1alpha1.ApplicationSpec{
						Chart: &v1alpha1.ChartRef{
							URL:     "http://stagingrepo",
							Name:    "appchart",
							Version: "0.1.6",
						},
						Release: &v1alpha1.Release{
							Values: &apiextensionsv1.JSON{
								Raw: bytesRelValues,
							},
						},
						Subcharts: []v1alpha1.DAG{
							{
								Name:         "subchart-3",
								Dependencies: []string{"subchart-2"},
							},
							{
								Name:         "subchart-2",
								Dependencies: []string{"subchart-1"},
							},
							{
								Name:         "subchart-1",
								Dependencies: nil,
							},
						},
					},
				},
				subchartsStatus: map[string]v1alpha1.ChartStatus{
					"subchart-1": {
						Version: "1.0.0",
					},
					"subchart-2": {
						Version: "1.0.0",
					},
					"subchart-3": {
						Version: "1.0.3",
					},
				},
			},
			want: []v1alpha13.DAGTask{
				{
					Name:     "subchart-3",
					Template: "helmrelease-executor",
					Arguments: v1alpha13.Arguments{
						Parameters: []v1alpha13.Parameter{
							{
								Name: "helmrelease",
								Value: utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "1ccd4cae89-subchart-3",
										Labels: map[string]string{
											ChartLabelKey:  "application",
											OwnershipLabel: "application",
											HeritageLabel:  "orkestra",
										},
										Annotations: map[string]string{
											v1alpha1.ParentChartAnnotation: "application",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   "1ccd4cae89-subchart-3",
												Version: "1.0.3",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: "subchart-3",
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"sc3-key":"sc3-value"}`),
										},
									},
								}),
							},
							{
								Name:  "timeout",
								Value: utils.ToAnyStringPtr("5m"),
							},
						},
					},
					Dependencies: []string{"subchart-2"},
				},
				{
					Name:     "subchart-2",
					Template: "helmrelease-executor",
					Arguments: v1alpha13.Arguments{
						Parameters: []v1alpha13.Parameter{
							{
								Name: "helmrelease",
								Value: utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "1ccd4cae89-subchart-2",
										Labels: map[string]string{
											ChartLabelKey:  "application",
											OwnershipLabel: "application",
											HeritageLabel:  "orkestra",
										},
										Annotations: map[string]string{
											v1alpha1.ParentChartAnnotation: "application",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   "1ccd4cae89-subchart-2",
												Version: "1.0.0",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: "subchart-2",
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"sc2-key":"sc2-value"}`),
										},
									},
								}),
							},
							{
								Name:  "timeout",
								Value: utils.ToAnyStringPtr("5m"),
							},
						},
					},
					Dependencies: []string{"subchart-1"},
				},
				{
					Name:     "subchart-1",
					Template: "helmrelease-executor",
					Arguments: v1alpha13.Arguments{
						Parameters: []v1alpha13.Parameter{
							{
								Name: "helmrelease",
								Value: utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "1ccd4cae89-subchart-1",
										Labels: map[string]string{
											ChartLabelKey:  "application",
											OwnershipLabel: "application",
											HeritageLabel:  "orkestra",
										},
										Annotations: map[string]string{
											v1alpha1.ParentChartAnnotation: "application",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   "1ccd4cae89-subchart-1",
												Version: "1.0.0",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: "subchart-1",
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"sc1-key":"sc1-value"}`),
										},
									},
								}),
							},
							{
								Name:  "timeout",
								Value: utils.ToAnyStringPtr("5m"),
							},
						},
					},
					Dependencies: []string{},
				},
				{
					Name:     "application",
					Template: "helmrelease-executor",
					Arguments: v1alpha13.Arguments{
						Parameters: []v1alpha13.Parameter{
							{
								Name: "helmrelease",
								Value: utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{
									TypeMeta: v1.TypeMeta{
										Kind:       "HelmRelease",
										APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
									},
									ObjectMeta: v1.ObjectMeta{
										Name: "application",
										Labels: map[string]string{
											ChartLabelKey:  "application",
											OwnershipLabel: "application",
											HeritageLabel:  "orkestra",
										},
									},
									Spec: fluxhelmv2beta1.HelmReleaseSpec{
										Chart: fluxhelmv2beta1.HelmChartTemplate{
											Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
												Chart:   "appchart",
												Version: "0.1.6",
												SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
													Kind:      "HelmRepository",
													Name:      "chartmuseum",
													Namespace: "testorkestra",
												},
											},
										},
										ReleaseName: "application",
										Values: &apiextensionsv1.JSON{
											Raw: []byte(`{"global":{"keyG":"valueG"},"subchart-1":{"enabled":false},"subchart-2":{"enabled":false},"subchart-3":{"enabled":false}}`),
										},
									},
								}),
							},
							{
								Name:  "timeout",
								Value: utils.ToAnyStringPtr("5m"),
							},
						},
					},
					Dependencies: []string{"subchart-3", "subchart-2", "subchart-1"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateSubchartAndAppDAGTasks(tt.args.appGroupName, tt.args.namespace, tt.args.app, tt.args.subchartsStatus)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateSubchartAndAppDAGTasks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("generateSubchartAndAppDAGTasks() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_appDAGTaskBuilder(t *testing.T) {
	type args struct {
		name         string
		templateName string
		timeout      *v1alpha13.AnyString
		hrStr        *v1alpha13.AnyString
	}
	tests := []struct {
		name string
		args args
		want v1alpha13.DAGTask
	}{
		{
			name: "testing with nil pointer args",
			args: args{
				name:         "myApp",
				templateName: "helmrelease-executor",
				timeout:      nil,
				hrStr:        nil,
			},
			want: v1alpha13.DAGTask{
				Name:     "myapp",
				Template: "helmrelease-executor",
				Arguments: v1alpha13.Arguments{
					Parameters: []v1alpha13.Parameter{
						{
							Name:  "helmrelease",
							Value: nil,
						},
						{
							Name:  "timeout",
							Value: nil,
						},
					},
				},
			},
		},
		{
			name: "testing with valid args",
			args: args{
				name:         "myApp",
				templateName: "helmrelease-executor",
				timeout:      utils.ToAnyStringPtr("5m"),
				hrStr:        utils.ToAnyStringPtr("empty"),
			},
			want: v1alpha13.DAGTask{
				Name:     "myapp",
				Template: "helmrelease-executor",
				Arguments: v1alpha13.Arguments{
					Parameters: []v1alpha13.Parameter{
						{
							Name:  "helmrelease",
							Value: utils.ToAnyStringPtr("empty"),
						},
						{
							Name:  "timeout",
							Value: utils.ToAnyStringPtr("5m"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appDAGTaskBuilder(tt.args.name, tt.args.templateName, tt.args.timeout, tt.args.hrStr)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("appDAGTaskBuilder() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_generateSubchartHelmRelease(t *testing.T) {
	type args struct {
		r            *v1alpha1.Release
		namespace    string
		appChartName string
		subchartName string
		version      string
	}
	tests := []struct {
		name    string
		args    args
		want    *fluxhelmv2beta1.HelmRelease
		wantErr bool
	}{
		{
			name: "testing with nil release",
			args: args{
				r:            nil,
				namespace:    "testorkestra",
				appChartName: "myAppchart",
				subchartName: "mySubchart",
				version:      "0.1.0",
			},
			want: &fluxhelmv2beta1.HelmRelease{
				TypeMeta: v1.TypeMeta{
					Kind:       "HelmRelease",
					APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "09eeec4d52-mysubchart",
				},
				Spec: fluxhelmv2beta1.HelmReleaseSpec{
					Chart: fluxhelmv2beta1.HelmChartTemplate{
						Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
							Chart:   "09eeec4d52-mysubchart",
							Version: "0.1.0",
							SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "chartmuseum",
								Namespace: "testorkestra",
							},
						},
					},
					ReleaseName: "mysubchart",
					Values: &apiextensionsv1.JSON{
						Raw: []byte(`{}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "testing with release values",
			args: args{
				r: &v1alpha1.Release{
					Values: &apiextensionsv1.JSON{
						Raw: []byte(`{"mySubchart":{"KeySC":"valueSC"}}`),
					},
				},
				namespace:    "testorkestra",
				appChartName: "myAppchart",
				subchartName: "mySubchart",
				version:      "0.1.0",
			},
			want: &fluxhelmv2beta1.HelmRelease{
				TypeMeta: v1.TypeMeta{
					Kind:       "HelmRelease",
					APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "09eeec4d52-mysubchart",
				},
				Spec: fluxhelmv2beta1.HelmReleaseSpec{
					Chart: fluxhelmv2beta1.HelmChartTemplate{
						Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
							Chart:   "09eeec4d52-mysubchart",
							Version: "0.1.0",
							SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "chartmuseum",
								Namespace: "testorkestra",
							},
						},
					},
					ReleaseName: "mysubchart",
					Values: &apiextensionsv1.JSON{
						Raw: []byte(`{"KeySC":"valueSC"}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "testing with release values of unknown subchart",
			args: args{
				r: &v1alpha1.Release{
					Values: &apiextensionsv1.JSON{
						Raw: []byte(`{"unknownSubchart":{"KeySC":"valueSC"}}`),
					},
				},
				namespace:    "testorkestra",
				appChartName: "myAppchart",
				subchartName: "mySubchart",
				version:      "0.1.0",
			},
			want: &fluxhelmv2beta1.HelmRelease{
				TypeMeta: v1.TypeMeta{
					Kind:       "HelmRelease",
					APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "09eeec4d52-mysubchart",
				},
				Spec: fluxhelmv2beta1.HelmReleaseSpec{
					Chart: fluxhelmv2beta1.HelmChartTemplate{
						Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
							Chart:   "09eeec4d52-mysubchart",
							Version: "0.1.0",
							SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "chartmuseum",
								Namespace: "testorkestra",
							},
						},
					},
					ReleaseName: "mysubchart",
					Values: &apiextensionsv1.JSON{
						Raw: []byte(`{}`),
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateSubchartHelmRelease(tt.args.r, tt.args.namespace, tt.args.appChartName, tt.args.subchartName, tt.args.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateSubchartHelmRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(got, tt.want) {
				t.Errorf("generateSubchartHelmRelease() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_helmReleaseBuilder(t *testing.T) {
	type args struct {
		r           *v1alpha1.Release
		namespace   string
		objMetaName string
		chName      string
		releaseName string
		version     string
	}
	tests := []struct {
		name string
		args args
		want *fluxhelmv2beta1.HelmRelease
	}{
		{
			name: "testing with nil release",
			args: args{
				r:           nil,
				namespace:   "testorkestra",
				objMetaName: "objMetaMychart",
				chName:      "myAppChart",
				releaseName: "myAppChartRelease",
				version:     "0.1.0",
			},
			want: &fluxhelmv2beta1.HelmRelease{
				TypeMeta: v1.TypeMeta{
					Kind:       "HelmRelease",
					APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "objmetamychart",
				},
				Spec: fluxhelmv2beta1.HelmReleaseSpec{
					Chart: fluxhelmv2beta1.HelmChartTemplate{
						Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
							Chart:   "myappchart",
							Version: "0.1.0",
							SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "chartmuseum",
								Namespace: "testorkestra",
							},
						},
					},
					ReleaseName: "myappchartrelease",
				},
			},
		},
		{
			name: "testing with valid release",
			args: args{
				r: &v1alpha1.Release{
					TargetNamespace: "targetOrkestra",
				},
				namespace:   "testorkestra",
				objMetaName: "objMetaMychart",
				chName:      "myAppChart",
				releaseName: "myAppChartRelease",
				version:     "0.1.0",
			},
			want: &fluxhelmv2beta1.HelmRelease{
				TypeMeta: v1.TypeMeta{
					Kind:       "HelmRelease",
					APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      "objmetamychart",
					Namespace: "targetOrkestra",
				},
				Spec: fluxhelmv2beta1.HelmReleaseSpec{
					Chart: fluxhelmv2beta1.HelmChartTemplate{
						Spec: fluxhelmv2beta1.HelmChartTemplateSpec{
							Chart:   "myappchart",
							Version: "0.1.0",
							SourceRef: fluxhelmv2beta1.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      "chartmuseum",
								Namespace: "testorkestra",
							},
						},
					},
					ReleaseName:     "myappchartrelease",
					TargetNamespace: "targetOrkestra",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := helmReleaseBuilder(tt.args.r, tt.args.namespace, tt.args.objMetaName, tt.args.chName, tt.args.releaseName, tt.args.version)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("helmReleaseBuilder() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_subchartValues(t *testing.T) {
	type args struct {
		sc string
		av apiextensionsv1.JSON
	}
	tests := []struct {
		name string
		args args
		want apiextensionsv1.JSON
	}{
		{
			name: "withGlobalSuchart",
			args: args{
				sc: "subchart",
				av: apiextensionsv1.JSON{
					Raw: []byte(`{"global":{"keyG":"valueG"},"subchart":{"keySC":"valueSC"}}`),
				},
			},
			want: apiextensionsv1.JSON{
				Raw: []byte(`{"global":{"keyG":"valueG"},"keySC":"valueSC"}`),
			},
		},
		{
			name: "withOnlyGlobal",
			args: args{
				sc: "subchart",
				av: apiextensionsv1.JSON{
					Raw: []byte(`{"global":{"keyG":"valueG"}}`),
				},
			},
			want: apiextensionsv1.JSON{
				Raw: []byte(`{"global":{"keyG":"valueG"}}`),
			},
		},
		{
			name: "withOnlySubchart",
			args: args{
				sc: "subchart",
				av: apiextensionsv1.JSON{
					Raw: []byte(`{"subchart":{"keySC":"valueSC"}}`),
				},
			},
			want: apiextensionsv1.JSON{
				Raw: []byte(`{"keySC":"valueSC"}`),
			},
		},
		{
			name: "withNone",
			args: args{
				sc: "subchart",
				av: apiextensionsv1.JSON{
					Raw: []byte(""),
				},
			},
			want: apiextensionsv1.JSON{
				Raw: []byte("{}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := make(map[string]interface{})
			_ = json.Unmarshal(tt.args.av.Raw, &values)
			if got, _ := subchartValues(tt.args.sc, values); !cmp.Equal(*got, tt.want) {
				t.Errorf("subchartValues() = %v", cmp.Diff(*got, tt.want))
			}
		})
	}
}
