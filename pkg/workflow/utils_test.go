package workflow

import (
	"os"
	"testing"
	"time"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getTimeout(t *testing.T) {
	type args struct {
		duration *metav1.Duration
	}
	tests := []struct {
		name string
		args args
		want *v1alpha13.AnyString
	}{
		{
			name: "testing nil",
			args: args{
				duration: nil,
			},
			want: utils.ToAnyStringPtr("5m"),
		},
		{
			name: "testing default metav1.Duration",
			args: args{
				duration: &metav1.Duration{},
			},
			want: utils.ToAnyStringPtr("0s"),
		},
		{
			name: "testing non nil",
			args: args{
				duration: &metav1.Duration{
					Duration: time.Minute,
				},
			},
			want: utils.ToAnyStringPtr("1m0s"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTimeout(tt.args.duration); *got != *tt.want {
				t.Errorf("getTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNamespace(t *testing.T) {
	tests := []struct {
		name string
		want string
		env  string
	}{
		{
			name: "testing WORKFLOW_NAMESPACE env not set",
			want: "orkestra",
			env:  "",
		},
		{
			name: "testing WORKFLOW_NAMESPACE env is set",
			want: "testorkestra",
			env:  "testorkestra",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				os.Setenv("WORKFLOW_NAMESPACE", tt.want)
			}
			if got := GetNamespace(); got != tt.want {
				t.Errorf("GetNamespace() = %v, want %v", got, tt.want)
			}
			if tt.env != "" {
				os.Unsetenv("WORKFLOW_NAMESPACE")
			}
		})
	}
}

func Test_workflowServiceAccountName(t *testing.T) {
	tests := []struct {
		name string
		want string
		env  string
	}{
		{
			name: "testing WORKFLOW_SERVICEACCOUNT_NAME env not set",
			want: "orkestra",
			env:  "",
		},
		{
			name: "testing WORKFLOW_SERVICEACCOUNT_NAME env is set",
			want: "orkestra_service_account",
			env:  "orkestra_service_account",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				os.Setenv("WORKFLOW_SERVICEACCOUNT_NAME", tt.want)
			}
			if got := workflowServiceAccountName(); got != tt.want {
				t.Errorf("workflowServiceAccountName() = %v, want %v", got, tt.want)
			}
			if tt.env != "" {
				os.Unsetenv("WORKFLOW_SERVICEACCOUNT_NAME")
			}
		})
	}
}
