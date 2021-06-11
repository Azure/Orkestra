package workflow

import (
	"context"
	"fmt"

	v1alpha12 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/go-logr/logr"
)

type ClientType string

type ExecutorFunc func(string, ExecutorAction) v1alpha12.Template

const (
	Forward ClientType = "forward"
	Reverse ClientType = "reverse"
)

var _ = ForwardWorkflowClient{}
var _ = ReverseWorkflowClient{}

type Client interface {
	// Generate the object required by the workflow engine
	Generate() error
	// Submit the object required by the workflow engine generated by the Generate method
	Submit(ctx context.Context) error

	// GetLogger returns the logger associated with the workflow client
	GetLogger() logr.Logger

	// GetWorkflow returns the workflow from the k8s apiserver associated with the workflow client
	GetWorkflow(context.Context) (*v1alpha12.Workflow, error)

	// GetClient returns the k8s client associated with the workflow
	GetClient() client.Client
}

type ClientOptions struct {
	parallelism *int64
	stagingRepo string
	namespace   string
}

type Builder struct {
	client     client.Client
	logger     logr.Logger
	clientType ClientType
	options    ClientOptions
	executor   ExecutorFunc

	nodes           map[string]v1alpha12.NodeStatus
	forwardWorkflow *v1alpha12.Workflow
	appGroup        *v1alpha1.ApplicationGroup
}

type ForwardWorkflowClient struct {
	client.Client
	logr.Logger
	ClientOptions
	executor ExecutorFunc

	workflow *v1alpha12.Workflow
	appGroup *v1alpha1.ApplicationGroup
}

type ReverseWorkflowClient struct {
	client.Client
	logr.Logger
	ClientOptions
	executor ExecutorFunc

	nodes           map[string]v1alpha12.NodeStatus
	forwardWorkflow *v1alpha12.Workflow
	reverseWorkflow *v1alpha12.Workflow
}

func NewBuilder(client client.Client, logger logr.Logger) *Builder {
	return &Builder{
		client:  client,
		logger:  logger,
		options: ClientOptions{},
	}
}

func (builder *Builder) Forward(appGroup *v1alpha1.ApplicationGroup) *Builder {
	builder.clientType = Forward
	builder.appGroup = appGroup
	return builder
}

func (builder *Builder) Reverse(forwardWorkflow *v1alpha12.Workflow, nodes map[string]v1alpha12.NodeStatus) *Builder {
	builder.clientType = Reverse
	builder.forwardWorkflow = forwardWorkflow
	builder.nodes = nodes
	return builder
}

func (builder *Builder) WithParallelism(numNodes int64) *Builder {
	builder.options.parallelism = &numNodes
	return builder
}

func (builder *Builder) WithStagingRepo(stagingURL string) *Builder {
	builder.options.stagingRepo = stagingURL
	return builder
}

func (builder *Builder) InNamespace(namespace string) *Builder {
	builder.options.namespace = namespace
	return builder
}

func (builder *Builder) WithExecutor(executor ExecutorFunc) *Builder {
	builder.executor = executor
	return builder
}

func (builder *Builder) Build() (Client, error) {
	switch builder.clientType {
	case Forward:
		forwardClient := &ForwardWorkflowClient{
			Client:        builder.client,
			Logger:        builder.logger,
			ClientOptions: builder.options,
			appGroup:      builder.appGroup,
			executor:      builder.executor,
		}
		if builder.executor == nil {
			forwardClient.executor = defaultExecutor
		}
		return forwardClient, nil
	case Reverse:
		reverseClient := &ReverseWorkflowClient{
			Client:          builder.client,
			Logger:          builder.logger,
			ClientOptions:   builder.options,
			forwardWorkflow: builder.forwardWorkflow,
			nodes:           builder.nodes,
			executor:        builder.executor,
		}
		if builder.executor == nil {
			reverseClient.executor = defaultExecutor
		}
		return reverseClient, nil
	}
	return nil, fmt.Errorf("failed to build engine because type wasn't specified")
}

func Run(ctx context.Context, wfClient Client) error {
	if err := wfClient.Generate(); err != nil {
		wfClient.GetLogger().Error(err, "engine failed to generate workflow")
		return fmt.Errorf("failed to generate workflow : %w", err)
	}
	if err := wfClient.Submit(ctx); err != nil {
		wfClient.GetLogger().Error(err, "engine failed to submit reverse workflow")
		return err
	}
	return nil
}

func Suspend(ctx context.Context, wfClient Client) error {
	// suspend a workflow if it is not already finished or suspended
	workflow, err := wfClient.GetWorkflow(ctx)
	if client.IgnoreNotFound(err) != nil {
		return err
	} else if err != nil || !workflow.Status.FinishedAt.IsZero() {
		wfClient.GetLogger().Info("forward workflow not found, no need to suspend")
		return nil
	}
	if workflow.Spec.Suspend == nil || !*workflow.Spec.Suspend {
		wfClient.GetLogger().Info("suspending the workflow")
		wfPatch := client.MergeFrom(workflow.DeepCopy())
		suspend := true
		workflow.Spec.Suspend = &suspend
		if err := wfClient.GetClient().Patch(ctx, workflow, wfPatch); err != nil {
			wfClient.GetLogger().Error(err, "failed to patch workflow")
			return err
		}
	}
	return nil
}

func GetNodes(wf *v1alpha12.Workflow) map[string]v1alpha12.NodeStatus {
	nodes := make(map[string]v1alpha12.NodeStatus)
	for _, node := range wf.Status.Nodes {
		nodes[node.ID] = node
	}
	return nodes
}

func initWorkflowObject(name, namespace string, parallelism *int64) *v1alpha12.Workflow {
	return &v1alpha12.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{HeritageLabel: Project},
		},
		TypeMeta: v1.TypeMeta{
			APIVersion: v1alpha12.WorkflowSchemaGroupVersionKind.GroupVersion().String(),
			Kind:       v1alpha12.WorkflowSchemaGroupVersionKind.Kind,
		},
		Spec: v1alpha12.WorkflowSpec{
			Entrypoint:  EntrypointTemplateName,
			Templates:   make([]v1alpha12.Template, 0),
			Parallelism: parallelism,
			PodGC: &v1alpha12.PodGC{
				Strategy: v1alpha12.PodGCOnWorkflowCompletion,
			},
		},
	}
}

func updateWorkflowTemplates(wf *v1alpha12.Workflow, tpls ...v1alpha12.Template) {
	wf.Spec.Templates = append(wf.Spec.Templates, tpls...)
}
