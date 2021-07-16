package workflow

import (
	"reflect"
	"testing"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetDiff(t *testing.T) {
	type args struct {
		appGroup         *v1alpha1.ApplicationGroup
		rollbackAppGroup *v1alpha1.ApplicationGroup
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "same set of applications",
			args: args{
				appGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup1",
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup2",
								},
							},
						},
					},
				},
				rollbackAppGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup1",
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup2",
								},
							},
						},
					},
				},
			},
			want: []string{},
		},
		{
			name: "more applications in the newer application group",
			args: args{
				appGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup1",
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup2",
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup3",
								},
							},
						},
					},
				},
				rollbackAppGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup1",
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup2",
								},
							},
						},
					},
				},
			},
			want: []string{"appgroup3"},
		},
		{
			name: "more applications in the rollback application group",
			args: args{
				appGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup1",
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup2",
								},
							},
						},
					},
				},
				rollbackAppGroup: &v1alpha1.ApplicationGroup{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: v1alpha1.ApplicationGroupSpec{
						Applications: []v1alpha1.Application{
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup1",
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup2",
								},
							},
							{
								DAG: v1alpha1.DAG{
									Name: "appgroup3",
								},
							},
						},
					},
				},
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wc := RollbackWorkflowClient{appGroup: tt.args.appGroup, rollbackAppGroup: tt.args.rollbackAppGroup}
			got := wc.getDiff()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Build() = %v, want %v", got, tt.want)
			}
		})
	}
}
