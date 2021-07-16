package controllers_test

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	bookinfo   = "bookinfo"
	ambassador = "ambassador"
	podinfo    = "podinfo"

	ambassadorChartURL        = "https://nitishm.github.io/charts"
	ambassadorOldChartVersion = "6.6.0"
	ambassadorChartVersion    = "6.7.9"

	bookinfoChartURL     = "https://nitishm.github.io/charts"
	bookinfoChartVersion = "v2"

	podinfoChartURL     = "https://stefanprodan.github.io/podinfo"
	podinfoChartVersion = "5.2.1"
)

var (
	defaultDuration = metav1.Duration{Duration: time.Minute * 5}     // treat as const
	letterRunes     = []rune("abcdefghijklmnopqrstuvwxyz1234567890") // treat as const
)

func isAllHelmReleasesInReadyState(helmReleases []fluxhelmv2beta1.HelmRelease) bool {
	allReady := true
	for _, release := range helmReleases {
		condition := meta.GetResourceCondition(&release, meta.ReadyCondition)
		if condition.Reason == meta.SucceededReason {
			allReady = false
		}
	}
	return allReady
}

func addApplication(appGroup v1alpha1.ApplicationGroup, app v1alpha1.Application) v1alpha1.ApplicationGroup {
	appGroup.Spec.Applications = append(appGroup.Spec.Applications, app)
	return appGroup
}

func defaultAppGroup(groupName, groupNamespace, targetNamespace string) *v1alpha1.ApplicationGroup {
	g := &v1alpha1.ApplicationGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      groupName,
			Namespace: groupNamespace,
		},
	}
	g.Spec.Applications = make([]v1alpha1.Application, 0)
	g.Spec.Applications = append(g.Spec.Applications, bookinfoApplication(targetNamespace, ambassador), ambassadorApplication(targetNamespace))
	return g
}

func smallAppGroup(groupName, groupNamespace, targetNamespace string) *v1alpha1.ApplicationGroup {
	g := &v1alpha1.ApplicationGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      groupName,
			Namespace: groupNamespace,
		},
	}
	g.Spec.Applications = make([]v1alpha1.Application, 0)
	g.Spec.Applications = append(g.Spec.Applications, podinfoApplication(targetNamespace))
	return g
}

func createUniqueAppGroupName(name string) string {
	return name + "-" + getRandomStringRunes(10)
}

func getRandomStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func boolToBoolPtr(in bool) *bool {
	return &in
}

func ambassadorApplication(targetNamespace string, dependencies ...string) v1alpha1.Application {
	values := []byte(fmt.Sprintf(`{
       "nameOverride": "%s",
	   "service": {
		  "type": "ClusterIP"
	   },
       "scope": {
          "singleNamespace": true
       }
	}`, targetNamespace))
	return v1alpha1.Application{
		DAG: v1alpha1.DAG{
			Name:         ambassador,
			Dependencies: dependencies,
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				URL:     ambassadorChartURL,
				Name:    ambassador,
				Version: ambassadorChartVersion,
			},
			Release: &v1alpha1.Release{
				Timeout:         &metav1.Duration{Duration: time.Minute * 10},
				TargetNamespace: targetNamespace,
				Values: &apiextensionsv1.JSON{
					Raw: values,
				},
				Interval: defaultDuration,
			},
		},
	}
}

func bookinfoApplication(targetNamespace string, dependencies ...string) v1alpha1.Application {
	values := []byte(`{
		"productpage": {
			"replicaCount": 1
		},
		"details": {
			"replicaCount": 1
		},
		"reviews": {
			"replicaCount": 1
		},
		"ratings": {
			"replicaCount": 1
		}
	}`)
	return v1alpha1.Application{
		DAG: v1alpha1.DAG{
			Name:         bookinfo,
			Dependencies: dependencies,
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				URL:     bookinfoChartURL,
				Name:    bookinfo,
				Version: bookinfoChartVersion,
			},
			Release: &v1alpha1.Release{
				TargetNamespace: targetNamespace,
				Values: &apiextensionsv1.JSON{
					Raw: values,
				},
				Interval: defaultDuration,
			},
			Subcharts: []v1alpha1.DAG{
				{
					Name:         "productpage",
					Dependencies: []string{"reviews"},
				},
				{
					Name:         "reviews",
					Dependencies: []string{"details", "ratings"},
				},
				{
					Name:         "ratings",
					Dependencies: []string{},
				},
				{
					Name:         "details",
					Dependencies: []string{},
				},
			},
		},
	}
}

func podinfoApplication(targetNamespace string, dependencies ...string) v1alpha1.Application {
	return v1alpha1.Application{
		DAG: v1alpha1.DAG{
			Name:         podinfo,
			Dependencies: dependencies,
		},
		Spec: v1alpha1.ApplicationSpec{
			Chart: &v1alpha1.ChartRef{
				URL:     podinfoChartURL,
				Name:    podinfo,
				Version: podinfoChartVersion,
			},
			Release: &v1alpha1.Release{
				TargetNamespace: targetNamespace,
				Interval:        defaultDuration,
			},
		},
	}
}
