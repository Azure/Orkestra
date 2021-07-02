package workflow

import (
	"testing"

	"github.com/Azure/Orkestra/pkg/utils"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/google/go-cmp/cmp"
)

func TestBuild(t *testing.T) {
	type args struct {
		entry string
		nodes map[string]v1alpha13.NodeStatus
	}

	hrValue := utils.HrToB64AnyStringPtr(&fluxhelmv2beta1.HelmRelease{})

	nodesWithoutChildren := map[string]v1alpha13.NodeStatus{
		"node1": {ID: "node1", Type: "Pod", Children: nil, Inputs: &v1alpha13.Inputs{
			Parameters: []v1alpha13.Parameter{{Value: hrValue}},
		}},
		"node2": {ID: "node2", Type: "Pod", Children: nil, Inputs: &v1alpha13.Inputs{
			Parameters: []v1alpha13.Parameter{{Value: hrValue}},
		}},
	}

	nodesWithChildren := map[string]v1alpha13.NodeStatus{
		"node1": {
			ID:       "node1",
			Type:     "Pod",
			Children: []string{"node2", "node3"},
			Inputs:   &v1alpha13.Inputs{Parameters: []v1alpha13.Parameter{{Value: hrValue}}},
		},
		"node2": {
			ID:       "node2",
			Type:     "Pod",
			Children: []string{"node3", "node4", "node5"},
			Inputs:   &v1alpha13.Inputs{Parameters: []v1alpha13.Parameter{{Value: hrValue}}},
		},
		"node3": {
			ID:       "node3",
			Type:     "Pod",
			Children: []string{"node6", "node7"},
			Inputs:   &v1alpha13.Inputs{Parameters: []v1alpha13.Parameter{{Value: hrValue}}},
		},
		"node4": {
			ID:       "node4",
			Type:     "Pod",
			Children: nil,
			Inputs:   &v1alpha13.Inputs{Parameters: []v1alpha13.Parameter{{Value: hrValue}}},
		},
		"node5": {
			ID:       "node5",
			Type:     "Pod",
			Phase:    "Skipped",
			Children: nil,
			Inputs:   &v1alpha13.Inputs{Parameters: []v1alpha13.Parameter{{Value: hrValue}}},
		},
		"node6": {
			ID:       "node6",
			Type:     "Pod",
			Phase:    "Failed",
			Children: nil,
			Inputs:   &v1alpha13.Inputs{Parameters: []v1alpha13.Parameter{{Value: hrValue}}},
		},
		"node7": {
			ID:       "node7",
			Type:     "Pod",
			Phase:    "Error",
			Children: nil,
			Inputs:   &v1alpha13.Inputs{Parameters: []v1alpha13.Parameter{{Value: hrValue}}},
		},
	}

	tests := []struct {
		name string
		args args
		want *Graph
		err  error
	}{
		{
			name: "testing with nil nodes",
			args: args{
				entry: "",
				nodes: nil,
			},
			want: nil,
			err:  ErrNoNodesFound,
		},
		{
			name: "testing with unknown entry node",
			args: args{
				entry: "unknown",
				nodes: map[string]v1alpha13.NodeStatus{"node1": {ID: "node1", Children: nil}},
			},
			want: nil,
			err:  ErrEntryNodeNotFound,
		},
		{
			name: "testing with NodeStatus.Type != Pod",
			args: args{
				entry: "node1",
				nodes: map[string]v1alpha13.NodeStatus{
					"node1": {ID: "node1", Type: "notPod", Children: nil},
				},
			},
			want: &Graph{
				nodes: map[string]v1alpha13.NodeStatus{
					"node1": {ID: "node1", Type: "notPod", Children: nil},
				},
				releases: map[int][]fluxhelmv2beta1.HelmRelease{},
				maxLevel: 1,
			},
			err: nil,
		},
		{
			name: "testing with nil NodeStatus.Inputs",
			args: args{
				entry: "node1",
				nodes: map[string]v1alpha13.NodeStatus{
					"node1": {ID: "node1", Type: "Pod", Children: nil},
				},
			},
			want: nil,
			err:  ErrInvalidInputsPtr,
		},
		{
			name: "testing with nil NodeStatus.Inputs.Parameters",
			args: args{
				entry: "node1",
				nodes: map[string]v1alpha13.NodeStatus{
					"node1": {ID: "node1", Type: "Pod", Children: nil, Inputs: &v1alpha13.Inputs{}},
				},
			},
			want: nil,
			err:  ErrNilParametersSlice,
		},
		{
			name: "testing with nil NodeStatus.Inputs.Parameters[0].Value",
			args: args{
				entry: "node1",
				nodes: map[string]v1alpha13.NodeStatus{
					"node1": {ID: "node1", Type: "Pod", Children: nil, Inputs: &v1alpha13.Inputs{
						Parameters: []v1alpha13.Parameter{{Name: "param"}},
					}},
				},
			},
			want: nil,
			err:  ErrInvalidValuePtr,
		},
		{
			name: "testing with no children",
			args: args{
				entry: "node2",
				nodes: nodesWithoutChildren,
			},
			want: &Graph{
				nodes: nodesWithoutChildren,
				releases: map[int][]fluxhelmv2beta1.HelmRelease{
					0: {{}},
				},
				maxLevel: 1,
			},
			err: nil,
		},
		{
			name: "testing with children, indirect children, and failed nodes",
			args: args{
				entry: "node1",
				nodes: nodesWithChildren,
			},
			want: &Graph{
				nodes: nodesWithChildren,
				releases: map[int][]fluxhelmv2beta1.HelmRelease{
					0: {{}},
					1: {{}},
					2: {{}, {}},
				},
				maxLevel: 4,
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Build(tt.args.entry, tt.args.nodes)
			if err != tt.err {
				t.Errorf("Build() error = %v, want %v", err, tt.err)
			}
			if !cmp.Equal(got, tt.want, cmp.AllowUnexported(Graph{})) {
				t.Errorf("Build() = %v", cmp.Diff(got, tt.want, cmp.AllowUnexported(Graph{})))
			}
		})
	}
}

func Test_isChild(t *testing.T) {
	type args struct {
		nodeID string
		node   v1alpha13.NodeStatus
	}
	tests := []struct {
		name  string
		graph Graph
		args  args
		want  bool
	}{
		{
			name: "testing not a child",
			graph: Graph{
				nodes: map[string]v1alpha13.NodeStatus{},
			},
			args: args{
				nodeID: "unknown",
				node:   v1alpha13.NodeStatus{ID: "node1"},
			},
			want: false,
		},
		{
			name: "testing node3 child of node2",
			graph: Graph{
				nodes: map[string]v1alpha13.NodeStatus{
					"node1": {ID: "node1", Children: nil},
					"node3": {ID: "node3", Children: nil},
				},
			},
			args: args{
				nodeID: "node3",
				node:   v1alpha13.NodeStatus{ID: "node2", Children: []string{"node1", "node3"}},
			},
			want: true,
		},
		{
			name: "testing node3 child of the child of node2",
			graph: Graph{
				nodes: map[string]v1alpha13.NodeStatus{
					"node1": {ID: "node1", Children: []string{"node3"}},
					"node3": {ID: "node3", Children: nil},
				},
			},
			args: args{
				nodeID: "node3",
				node:   v1alpha13.NodeStatus{ID: "node2", Children: []string{"node1"}},
			},
			want: true,
		},
		{
			name: "testing child node not in the graph",
			graph: Graph{
				nodes: map[string]v1alpha13.NodeStatus{
					"node1": {ID: "node1", Children: nil},
				},
			},
			args: args{
				nodeID: "node3",
				node:   v1alpha13.NodeStatus{ID: "node2", Children: []string{"node1", "node3"}},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.graph.isChild(tt.args.nodeID, tt.args.node); got != tt.want {
				t.Errorf("Graph.isChild() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		name  string
		graph Graph
		want  [][]fluxhelmv2beta1.HelmRelease
	}{
		{
			name: "testing with empty releases",
			graph: Graph{
				nodes:    nil,
				releases: map[int][]fluxhelmv2beta1.HelmRelease{},
				maxLevel: 4,
			},
			want: [][]fluxhelmv2beta1.HelmRelease{},
		},
		{
			name: "testing with valid input",
			graph: Graph{
				nodes: nil,
				releases: map[int][]fluxhelmv2beta1.HelmRelease{
					0: {{}},
					1: {{}},
					2: {{}, {}},
				},
				maxLevel: 4,
			},
			want: [][]fluxhelmv2beta1.HelmRelease{
				{{}, {}},
				{{}},
				{{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.graph.Reverse(); !cmp.Equal(got, tt.want) {
				t.Errorf("Build() = %v", cmp.Diff(got, tt.want))
			}
		})
	}
}
