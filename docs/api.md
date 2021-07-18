---
layout: default
title: API Reference 
nav_order: 3
---
<h1>Orkestra API Reference</h1>
<p>Packages:</p>
<ul class="simple">
<li>
<a href="#orkestra.azure.microsoft.com%2fv1alpha1">orkestra.azure.microsoft.com/v1alpha1</a>
</li>
</ul>
<h2 id="orkestra.azure.microsoft.com/v1alpha1">orkestra.azure.microsoft.com/v1alpha1</h2>
<p>Package v1alpha1 contains API Schema definitions for the Orkestra v1alpha1 API group.</p>
<h3>Resource Types:</h3>
<ul class="simple"></ul>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.Application">Application
</h3>
<p>
(<em>Appears on:</em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationGroupSpec">ApplicationGroupSpec</a>)
</p>
<p>Application spec and dependency on other applications</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>DAG</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.DAG">
DAG
</a>
</em>
</td>
<td>
<p>
(Members of <code>DAG</code> are embedded into this type.)
</p>
<p>DAG contains the dependency information</p>
</td>
</tr>
<tr>
<td>
<code>spec</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationSpec">
ApplicationSpec
</a>
</em>
</td>
<td>
<p>Spec contains the application spec including the chart info and overlay values</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>chart</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ChartRef">
ChartRef
</a>
</em>
</td>
<td>
<p>Chart holds the values needed to pull the chart</p>
</td>
</tr>
<tr>
<td>
<code>release</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.Release">
Release
</a>
</em>
</td>
<td>
<p>Release holds the values to apply to the helm release</p>
</td>
</tr>
<tr>
<td>
<code>executor</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.Executor">
Executor
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Executor holds container customizations for the executor to use when running pods</p>
</td>
</tr>
<tr>
<td>
<code>subcharts</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.DAG">
[]DAG
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subcharts provides the dependency order among the subcharts of the application</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.ApplicationGroup">ApplicationGroup
</h3>
<p>ApplicationGroup is the Schema for the applicationgroups API</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationGroupSpec">
ApplicationGroupSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>applications</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.Application">
[]Application
</a>
</em>
</td>
<td>
<p>Applications that make up the application group</p>
</td>
</tr>
<tr>
<td>
<code>interval</code><br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Interval specifies the between reconciliations of the ApplicationGroup
Defaults to 5s for short requeue and 30s for long requeue</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationGroupStatus">
ApplicationGroupStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.ApplicationGroupSpec">ApplicationGroupSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationGroup">ApplicationGroup</a>)
</p>
<p>ApplicationGroupSpec defines the desired state of ApplicationGroup</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>applications</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.Application">
[]Application
</a>
</em>
</td>
<td>
<p>Applications that make up the application group</p>
</td>
</tr>
<tr>
<td>
<code>interval</code><br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Interval specifies the between reconciliations of the ApplicationGroup
Defaults to 5s for short requeue and 30s for long requeue</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.ApplicationGroupStatus">ApplicationGroupStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationGroup">ApplicationGroup</a>)
</p>
<p>ApplicationGroupStatus defines the observed state of ApplicationGroup</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>applications</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationStatus">
[]ApplicationStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Applications status</p>
</td>
</tr>
<tr>
<td>
<code>observedGeneration</code><br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ObservedGeneration captures the last generation
that was captured and completed by the reconciler</p>
</td>
</tr>
<tr>
<td>
<code>lastSucceededGeneration</code><br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>LastSucceededGeneration captures the last generation
that has successfully completed a full workflow rollout of the application group</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Conditions holds the conditions of the ApplicationGroup</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.ApplicationSpec">ApplicationSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.Application">Application</a>)
</p>
<p>ApplicationSpec defines the desired state of Application</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>chart</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ChartRef">
ChartRef
</a>
</em>
</td>
<td>
<p>Chart holds the values needed to pull the chart</p>
</td>
</tr>
<tr>
<td>
<code>release</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.Release">
Release
</a>
</em>
</td>
<td>
<p>Release holds the values to apply to the helm release</p>
</td>
</tr>
<tr>
<td>
<code>executor</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.Executor">
Executor
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Executor holds container customizations for the executor to use when running pods</p>
</td>
</tr>
<tr>
<td>
<code>subcharts</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.DAG">
[]DAG
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subcharts provides the dependency order among the subcharts of the application</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.ApplicationStatus">ApplicationStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationGroupStatus">ApplicationGroupStatus</a>)
</p>
<p>ApplicationStatus shows the current status of the application helm release</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Name of the application</p>
</td>
</tr>
<tr>
<td>
<code>ChartStatus</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ChartStatus">
ChartStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>ChartStatus</code> are embedded into this type.)
</p>
<em>(Optional)</em>
<p>ChartStatus for the application helm chart</p>
</td>
</tr>
<tr>
<td>
<code>subcharts</code><br>
<em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ChartStatus">
map[string]./api/v1alpha1.ChartStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Subcharts contains the subchart chart status</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.ChartRef">ChartRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationSpec">ApplicationSpec</a>)
</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>url</code><br>
<em>
string
</em>
</td>
<td>
<p>The Helm repository URL, a valid URL contains at least a protocol and host.</p>
</td>
</tr>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
<p>The name or path the Helm chart is available at in the SourceRef.</p>
</td>
</tr>
<tr>
<td>
<code>version</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Version semver expression, ignored for charts from v1beta1.GitRepository and
v1beta1.Bucket sources. Defaults to latest when omitted.</p>
</td>
</tr>
<tr>
<td>
<code>authSecretRef</code><br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#objectreference-v1-core">
Kubernetes core/v1.ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AuthSecretRef is a reference to the auth secret
to access a private helm repository</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.ChartStatus">ChartStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationStatus">ApplicationStatus</a>)
</p>
<p>ChartStatus shows the current status of the Application Reconciliation process</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>error</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Error string from the error during reconciliation (if any)</p>
</td>
</tr>
<tr>
<td>
<code>version</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Version of the chart/subchart</p>
</td>
</tr>
<tr>
<td>
<code>staged</code><br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Staged if true denotes that the chart/subchart has been pushed to the
staging helm repo</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Conditions holds the conditions for the ChartStatus</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.DAG">DAG
</h3>
<p>
(<em>Appears on:</em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.Application">Application</a>, 
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationSpec">ApplicationSpec</a>)
</p>
<p>DAG contains the dependency information</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
<p>Name of the application</p>
</td>
</tr>
<tr>
<td>
<code>dependencies</code><br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Dependencies on other applications by name</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.Executor">Executor
</h3>
<p>
(<em>Appears on:</em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationSpec">ApplicationSpec</a>)
</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br>
<em>
string
</em>
</td>
<td>
<p>Name of the container</p>
</td>
</tr>
<tr>
<td>
<code>image</code><br>
<em>
string
</em>
</td>
<td>
<p>Docker image name</p>
</td>
</tr>
<tr>
<td>
<code>tag</code><br>
<em>
string
</em>
</td>
<td>
<p>Docker image tag</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.Release">Release
</h3>
<p>
(<em>Appears on:</em>
<a href="#orkestra.azure.microsoft.com/v1alpha1.ApplicationSpec">ApplicationSpec</a>)
</p>
<div class="md-typeset__scrollwrap">
<div class="md-typeset__table">
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>interval</code><br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Interval at which to reconcile the Helm release.</p>
</td>
</tr>
<tr>
<td>
<code>targetNamespace</code><br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TargetNamespace to target when performing operations for the HelmRelease.
Defaults to the namespace of the HelmRelease.</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code><br>
<em>
<a href="https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Timeout is the time to wait for any individual Kubernetes operation (like Jobs
for hooks) during the performance of a Helm action. Defaults to &lsquo;5m0s&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>values</code><br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1?tab=doc#JSON">
Kubernetes pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Values holds the values for this Helm release.</p>
</td>
</tr>
<tr>
<td>
<code>install</code><br>
<em>
<a href="https://pkg.go.dev/github.com/fluxcd/helm-controller/api/v2beta1#Install">
helm-controller v2beta1.Install
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Install holds the configuration for Helm install actions for this HelmRelease.</p>
</td>
</tr>
<tr>
<td>
<code>upgrade</code><br>
<em>
<a href="https://pkg.go.dev/github.com/fluxcd/helm-controller/api/v2beta1#Upgrade">
helm-controller v2beta1.Upgrade
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Upgrade holds the configuration for Helm upgrade actions for this HelmRelease.</p>
</td>
</tr>
<tr>
<td>
<code>rollback</code><br>
<em>
<a href="https://pkg.go.dev/github.com/fluxcd/helm-controller/api/v2beta1#Rollback">
helm-controller v2beta1.Rollback
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Rollback holds the configuration for Helm rollback actions for this HelmRelease.</p>
</td>
</tr>
<tr>
<td>
<code>uninstall</code><br>
<em>
<a href="https://pkg.go.dev/github.com/fluxcd/helm-controller/api/v2beta1#Uninstall">
helm-controller v2beta1.Uninstall
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Rollback holds the configuration for Helm uninstall actions for this HelmRelease.</p>
</td>
</tr>
</tbody>
</table>
</div>
</div>
<h3 id="orkestra.azure.microsoft.com/v1alpha1.WorkflowType">WorkflowType
(<code>string</code> alias)</h3>
<p class="last">This page was automatically generated with <code>gen-crd-api-reference-docs</code></p>
