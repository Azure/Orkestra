// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package v1alpha1

import (
	"encoding/json"
	"time"

	"github.com/Azure/Orkestra/pkg/meta"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkflowType string

const (
	Forward  WorkflowType = "forward"
	Reverse  WorkflowType = "reverse"
	Rollback WorkflowType = "rollback"
)

var WorkflowConditionMap = map[WorkflowType]string{
	Forward:  meta.ForwardWorkflowSucceededCondition,
	Reverse:  meta.ReverseWorkflowSucceededCondition,
	Rollback: meta.RollbackWorkflowSucceededCondition,
}

const (
	DefaultProgressingRequeue = 5 * time.Second
	DefaultSucceededRequeue   = 5 * time.Minute

	AppGroupNameKey   = "appgroup"
	AppGroupFinalizer = "application-group-finalizer"

	LastSuccessfulAnnotation = "orkestra.azure.microsoft.com/last-successful-applicationgroup"
	ParentChartAnnotation    = "orkestra.azure.microsoft.com/parent-chart"

	HeritageLabel = "orkestra.azure.microsoft.com/heritage"
	HeritageValue = "orkestra"

	OwnershipLabel = "orkestra.azure.microsoft.com/owner"
	ChartLabel     = "orkestra.azure.microsoft.com/chart"

	ForwardWorkflow  WorkflowType = "forward"
	ReverseWorkflow  WorkflowType = "reverse"
	RollbackWorkflow WorkflowType = "rollback"
)

// GetInterval returns the interval if specified in the application group
// Otherwise, it returns the default requeue time for the appGroup
func GetInterval(appGroup *ApplicationGroup) time.Duration {
	if appGroup.Spec.Interval != nil {
		return appGroup.Spec.Interval.Duration
	}
	return DefaultSucceededRequeue
}

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// Chart holds the values needed to pull the chart
	// +required
	Chart *ChartRef `json:"chart"`

	// Release holds the values to apply to the helm release
	// +required
	Release *Release `json:"release"`

	// Subcharts provides the dependency order among the subcharts of the application
	// +optional
	Subcharts []DAG `json:"subcharts,omitempty"`
}

type Release struct {
	// Interval at which to reconcile the Helm release.
	// +kubebuilder:default:="5m"
	// +optional
	Interval metav1.Duration `json:"interval,omitempty"`

	// TargetNamespace to target when performing operations for the HelmRelease.
	// Defaults to the namespace of the HelmRelease.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Optional
	// +optional
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Timeout is the time to wait for any individual Kubernetes operation (like Jobs
	// for hooks) during the performance of a Helm action. Defaults to '5m0s'.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Values holds the values for this Helm release.
	// +optional
	Values *apiextensionsv1.JSON `json:"values,omitempty"`

	// Install holds the configuration for Helm install actions for this HelmRelease.
	// +optional
	Install *fluxhelmv2beta1.Install `json:"install,omitempty"`

	// Upgrade holds the configuration for Helm upgrade actions for this HelmRelease.
	// +optional
	Upgrade *fluxhelmv2beta1.Upgrade `json:"upgrade,omitempty"`

	// Rollback holds the configuration for Helm rollback actions for this HelmRelease.
	// +optional
	Rollback *fluxhelmv2beta1.Rollback `json:"rollback,omitempty"`

	// Rollback holds the configuration for Helm uninstall actions for this HelmRelease.
	// +optional
	Uninstall *fluxhelmv2beta1.Uninstall `json:"uninstall,omitempty"`
}

type ChartRef struct {
	// The Helm repository URL, a valid URL contains at least a protocol and host.
	// +required
	URL string `json:"url"`

	// The name or path the Helm chart is available at in the SourceRef.
	// +required
	Name string `json:"name"` //nolint: golint

	// Version semver expression, ignored for charts from v1beta1.GitRepository and
	// v1beta1.Bucket sources. Defaults to latest when omitted.
	// +kubebuilder:default:=*
	// +optional
	Version string `json:"version,omitempty"`

	// AuthSecretRef is a reference to the auth secret
	// to access a private helm repository
	// +optional
	AuthSecretRef *corev1.ObjectReference `json:"authSecretRef,omitempty"`
}

// ChartStatus shows the current status of the Application Reconciliation process
type ChartStatus struct {
	// Error string from the error during reconciliation (if any)
	// +optional
	Error string `json:"error,omitempty"`

	// Version of the chart/subchart
	// +optional
	Version string `json:"version,omitempty"`

	// Staged if true denotes that the chart/subchart has been pushed to the
	// staging helm repo
	// +optional
	Staged bool `json:"staged,omitempty"`

	// +optional
	// Conditions holds the conditions for the ChartStatus
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ApplicationGroupSpec defines the desired state of ApplicationGroup
type ApplicationGroupSpec struct {
	// Applications that make up the application group
	// +kubebuilder:validation:MinItems:=1
	// +required
	Applications []Application `json:"applications,omitempty"`

	// Interval specifies the between reconciliations of the ApplicationGroup
	// Defaults to 5s for short requeue and 30s for long requeue
	// +optional
	Interval *metav1.Duration `json:"interval,omitempty"`
}

// Application spec and dependency on other applications
type Application struct {
	// DAG contains the dependency information
	DAG `json:",inline"`

	// Spec contains the application spec including the chart info and overlay values
	Spec ApplicationSpec `json:"spec,omitempty"`
}

// DAG contains the dependency information
type DAG struct {
	// Name of the application
	// +required
	Name string `json:"name"`

	// Dependencies on other applications by name
	// +optional
	Dependencies []string `json:"dependencies,omitempty"`
}

// ApplicationStatus shows the current status of the application helm release
type ApplicationStatus struct {
	// Name of the application
	// +optional
	Name string `json:"name"`

	// ChartStatus for the application helm chart
	// +optional
	ChartStatus `json:",inline"`

	// Subcharts contains the subchart chart status
	// +optional
	Subcharts map[string]ChartStatus `json:"subcharts,omitempty"`
}

// ApplicationGroupStatus defines the observed state of ApplicationGroup
type ApplicationGroupStatus struct {
	// Applications status
	// +optional
	Applications []ApplicationStatus `json:"applications,omitempty"`

	// ObservedGeneration captures the last generation
	// that was captured and completed by the reconciler
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastSucceededGeneration captures the last generation
	// that has successfully completed a full workflow rollout of the application group
	// +optional
	LastSucceededGeneration int64 `json:"lastSucceededGeneration,omitempty"`

	// Conditions holds the conditions of the ApplicationGroup
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// GetValues unmarshals the raw values to a map[string]interface{} and returns
// the result.
func (in *Release) GetValues() map[string]interface{} {
	values := make(map[string]interface{})
	if in.Values != nil {
		_ = json.Unmarshal(in.Values.Raw, &values)
	}
	return values
}

// GetValues unmarshals the raw values to a map[string]interface{} and returns
// the result.
func (in *Application) GetValues() map[string]interface{} {
	return in.Spec.Release.GetValues()
}

func GetJSON(values map[string]interface{}) (*apiextensionsv1.JSON, error) {
	bytes, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return &apiextensionsv1.JSON{
		Raw: bytes,
	}, nil
}

func (in *ApplicationGroup) GetApplicationNames() []string {
	var names []string
	for _, application := range in.Spec.Applications {
		names = append(names, application.Name)
	}
	return names
}

// SetValues marshals the raw values into the JSON values
func (in *Application) SetValues(values map[string]interface{}) error {
	bytes, err := json.Marshal(values)
	if err != nil {
		return err
	}
	in.Spec.Release.Values.Raw = bytes
	return nil
}

// ReadySucceeded sets the meta.ReadyCondition to 'True', with the given
// meta.Succeeded reason and message
func (in *ApplicationGroup) ReadySucceeded() {
	in.Status.LastSucceededGeneration = in.Generation
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionTrue, meta.SucceededReason, "workflow and reconciliation succeeded")
}

// WorkflowFailed sets the meta.ReadyCondition to 'False' and
// meta.ReadyWorkflowFailed reason and message
func (in *ApplicationGroup) WorkflowFailed(message string) {
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionFalse, meta.WorkflowFailedReason, message)
}

// ChartPullFailed sets the meta.ReadyCondition to 'False' and
// meta.ChartPullFailedReason reason and message
func (in *ApplicationGroup) ChartPullFailed(message string) {
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionFalse, meta.ChartPullFailedReason, message)
}

// WorkflowTemplateGenerationFailed sets the meta.ReadyCondition to 'False' and
// meta.TemplateGenerationFailed reason and message
func (in *ApplicationGroup) WorkflowTemplateGenerationFailed(message string) {
	meta.SetResourceCondition(in, meta.ReadyCondition, metav1.ConditionFalse, meta.WorkflowTemplateGenerationFailedReason, message)
}

// GetReadyCondition gets the string condition.Reason of the
// meta.ReadyCondition type
func (in *ApplicationGroup) GetReadyCondition() string {
	condition := meta.GetResourceCondition(in, meta.ReadyCondition)
	if condition == nil {
		return meta.ProgressingReason
	}
	return condition.Reason
}

// GetWorkflowCondition gets the string condition.Reason of the given workflow type
func (in *ApplicationGroup) GetWorkflowCondition(wfType WorkflowType) string {
	var condition *metav1.Condition
	if wfCondition, ok := WorkflowConditionMap[wfType]; ok {
		condition = meta.GetResourceCondition(in, wfCondition)
	}
	if condition == nil {
		return meta.ProgressingReason
	}
	return condition.Reason
}

// GetStatusConditions gets the status conditions from the
// ApplicationGroup status
func (in *ApplicationGroup) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// GetStatusConditions gets the status conditions from the
// ChartStatus status
func (in *ChartStatus) GetStatusConditions() *[]metav1.Condition {
	return &in.Conditions
}

func (in *ApplicationGroup) GetLastSuccessful() *ApplicationGroupSpec {
	lastSuccessful := &ApplicationGroupSpec{}
	if s, ok := in.Annotations[LastSuccessfulAnnotation]; ok {
		_ = json.Unmarshal([]byte(s), lastSuccessful)
		return lastSuccessful
	}
	return nil
}

func (in *ApplicationGroup) SetLastSuccessful() {
	b, _ := json.Marshal(&in.Spec)
	if in.Annotations == nil {
		in.Annotations = make(map[string]string)
	}
	in.Annotations[LastSuccessfulAnnotation] = string(b)
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=applicationgroups,scope=Cluster,shortName={"ag","appgroup"}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ApplicationGroup is the Schema for the applicationgroups API
type ApplicationGroup struct { //nolint: gocritic
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationGroupSpec   `json:"spec,omitempty"`
	Status ApplicationGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationGroupList contains a list of ApplicationGroup
type ApplicationGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationGroup{}, &ApplicationGroupList{})
}
