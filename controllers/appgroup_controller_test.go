package controllers_test

import (
	"context"
	"time"

	"github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/pkg/meta"

	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	meta2 "github.com/fluxcd/pkg/apis/meta"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("ApplicationGroup Controller", func() {
	const (
		defaultNamespace = "orkestra"
		defaultTimeout   = time.Minute * 5

		// Total number of helm releases in the default app group, including subcharts.
		totalHelmReleaseCount = 6

		// Total number of application helm releases in the default app group, excluding subcharts.
		onlyApplicationsHelmReleaseCount = 2
	)

	var (
		ctx      context.Context
		name     string
		appGroup *v1alpha1.ApplicationGroup
		key      types.NamespacedName
	)

	BeforeEach(func() {
		ctx = context.Background()
		name = createUniqueAppGroupName("bookinfo")
		appGroup = defaultAppGroup(name, defaultNamespace, name)
		key = client.ObjectKeyFromObject(appGroup)
	})

	AfterEach(func() {
		By("Cleanup: Deleting the bookinfo object from the cluster")
		patch := client.MergeFrom(appGroup.DeepCopy())
		controllerutil.RemoveFinalizer(appGroup, v1alpha1.AppGroupFinalizer)
		_ = k8sClient.Patch(ctx, appGroup, patch)
		_ = k8sClient.Delete(ctx, appGroup)

		By("Cleanup: Calling delete on the Helm Releases")
		_ = k8sClient.DeleteAllOf(ctx, &fluxhelmv2beta1.HelmRelease{}, client.InNamespace(name))
		_ = k8sClient.DeleteAllOf(ctx, &v1alpha13.Workflow{}, client.InNamespace(name))
	})

	It("Should create Bookinfo spec successfully", func() {
		By("Applying the bookinfo object to the cluster")
		err := k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Getting the Helm Release list")
		hrList := &fluxhelmv2beta1.HelmReleaseList{}
		err = k8sClient.List(ctx, hrList, client.InNamespace(name))
		Expect(err).ToNot(HaveOccurred())
		oldHelmReleaseCount := len(hrList.Items)

		By("Making sure that the workflow goes into a running state")
		Eventually(func() bool {
			wf := &v1alpha13.Workflow{}
			wfKey := types.NamespacedName{Name: name, Namespace: defaultNamespace}
			_ = k8sClient.Get(ctx, wfKey, wf)
			return string(wf.Status.Phase) == string(v1alpha13.NodeRunning)
		}, time.Minute, time.Second).Should(BeTrue())

		By("Waiting for the bookinfo object to reach a succeeded reason")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.SucceededReason
		}, defaultTimeout, time.Second).Should(BeTrue())

		By("Checking that all the HelmReleases have come up and are in a ready state")
		err = k8sClient.List(ctx, hrList, client.InNamespace(name))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(hrList.Items)).To(Equal(oldHelmReleaseCount + totalHelmReleaseCount))
		Expect(isAllHelmReleasesInReadyState(hrList.Items)).To(BeTrue())

		By("Waiting for all the HelmReleases to delete")
		err = k8sClient.Delete(ctx, appGroup)
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the Workflow to delete all the HelmReleases")
		Eventually(func() bool {
			helmReleases := &fluxhelmv2beta1.HelmReleaseList{}
			if err := k8sClient.List(ctx, helmReleases, client.InNamespace(name)); err != nil {
				return false
			}
			return len(helmReleases.Items) == 0
		}, defaultTimeout, time.Second).Should(BeTrue())
	})

	It("Should create only application releases with subchart nil successfully", func() {
		for i := range appGroup.Spec.Applications {
			appGroup.Spec.Applications[i].Spec.Subcharts = nil
		}

		By("Applying the bookinfo object to the cluster")
		err := k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Getting the Helm Release list")
		hrList := &fluxhelmv2beta1.HelmReleaseList{}
		err = k8sClient.List(ctx, hrList, client.InNamespace(name))
		Expect(err).ToNot(HaveOccurred())
		oldHelmReleaseCount := len(hrList.Items)

		By("Making sure that the workflow goes into a running state")
		Eventually(func() bool {
			wf := &v1alpha13.Workflow{}
			wfKey := types.NamespacedName{Name: name, Namespace: defaultNamespace}
			_ = k8sClient.Get(ctx, wfKey, wf)
			return string(wf.Status.Phase) == string(v1alpha13.NodeRunning)
		}, time.Minute, time.Second).Should(BeTrue())

		By("Waiting for the bookinfo object to reach a succeeded reason")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.SucceededReason
		}, defaultTimeout, time.Second).Should(BeTrue())

		By("Checking that the all the HelmReleases have come up and are in a ready state")
		err = k8sClient.List(ctx, hrList, client.InNamespace(name))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(hrList.Items)).To(Equal(oldHelmReleaseCount + onlyApplicationsHelmReleaseCount))
		Expect(isAllHelmReleasesInReadyState(hrList.Items)).To(BeTrue())
	})

	It("Should fail to create and post a failed error state", func() {
		appGroup.Spec.Applications[0].Spec.Chart.Version = "fake-version"

		By("Applying the bookinfo object to the cluster")
		err := k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo object to reach a chart pull failed reason")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			readyCondition := meta.GetResourceCondition(appGroup, meta.ReadyCondition)
			if readyCondition == nil {
				return false
			}
			return readyCondition.Reason == meta.ChartPullFailedReason
		}, time.Second*30, time.Second).Should(BeTrue())
	})

	It("Should create the bookinfo and then update it", func() {
		By("Applying the bookinfo object to the cluster")
		err := k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo object to reach a succeeded reason")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.SucceededReason
		}, defaultTimeout, time.Second).Should(BeTrue())

		By("Adding application to the AppGroup Spec after the AppGroup has fully reconciled")
		newAppGroup := addApplication(*appGroup, podinfoApplication(name))
		err = k8sClient.Update(ctx, &newAppGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo application group to reach a succeeded reason")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.SucceededReason && appGroup.Generation == appGroup.Status.ObservedGeneration
		}, time.Minute*2, time.Second).Should(BeTrue())
	})

	It("Should fail to install, then get updated and pass getting installed", func() {
		appGroup.Spec.Applications[0].Spec.Chart.Version = "fake-version"

		By("Applying the bookinfo object to the cluster")
		err := k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo object to go into a failed state")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			readyCondition := meta.GetResourceCondition(appGroup, meta.ReadyCondition)
			if readyCondition == nil {
				return false
			}
			return readyCondition.Reason == meta.ChartPullFailedReason
		}, time.Second*30, time.Second).Should(BeTrue())

		patch := client.MergeFrom(appGroup.DeepCopy())
		appGroup.Spec.Applications[0].Spec.Chart.Version = bookinfoChartVersion
		err = k8sClient.Patch(ctx, appGroup, patch)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo object to reach a succeeded deploy condition")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.ProgressingReason
		}, time.Minute*2, time.Second).Should(BeTrue())
	})

	It("Should succeed to upgrade the versions of helm releases to newer versions", func() {
		appGroup.Spec.Applications[1].Spec.Chart.Version = ambassadorOldChartVersion

		By("Applying the bookinfo object to the cluster")
		err := k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo object to reach a succeeded reason")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.SucceededReason
		}, defaultTimeout, time.Second).Should(BeTrue())

		By("Upgrading the charts to a newer version")
		patch := client.MergeFrom(appGroup.DeepCopy())
		appGroup.Spec.Applications[1].Spec.Chart.Version = ambassadorChartVersion
		err = k8sClient.Patch(ctx, appGroup, patch)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the application group to start its upgrade")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.ProgressingReason && appGroup.Generation > 1
		}, time.Second*30, time.Second).Should(BeTrue())

		By("Waiting for the newer version of the charts to be released")
		Eventually(func() bool {
			hr := &fluxhelmv2beta1.HelmRelease{}
			hrKey := types.NamespacedName{Name: ambassador, Namespace: name}
			if err := k8sClient.Get(ctx, hrKey, hr); err != nil {
				return false
			}
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return hr.Spec.Chart.Spec.Version == ambassadorChartVersion && appGroup.GetReadyCondition() == meta.SucceededReason
		}, defaultTimeout, time.Second).Should(BeTrue())

	})

	It("Should succeed to rollback helm chart versions on failure", func() {
		appGroup.Spec.Applications[1].Spec.Chart.Version = ambassadorOldChartVersion

		By("Applying the bookinfo object to the cluster")
		err := k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo object to reach a succeeded reason")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			_, exist := appGroup.Annotations[v1alpha1.LastSuccessfulAnnotation]
			return appGroup.GetReadyCondition() == meta.SucceededReason && exist
		}, defaultTimeout, time.Second).Should(BeTrue())

		By("Upgrading the ambassador chart to a newer version while intentionally timing out the last DAG step")
		patch := client.MergeFrom(appGroup.DeepCopy())
		appGroup.Spec.Applications[1].Spec.Chart.Version = ambassadorChartVersion
		appGroup.Spec.Applications[0].Spec.Release.Timeout = &metav1.Duration{Duration: time.Second}
		err = k8sClient.Patch(ctx, appGroup, patch)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the application group to start its upgrade")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.ProgressingReason && appGroup.Generation > 1
		}, time.Second*30, time.Second).Should(BeTrue())

		By("Waiting for the newer version of the charts to be released")
		Eventually(func() bool {
			hr := &fluxhelmv2beta1.HelmRelease{}
			hrKey := types.NamespacedName{Name: ambassador, Namespace: name}
			if err := k8sClient.Get(ctx, hrKey, hr); err != nil {
				return false
			}
			return hr.Spec.Chart.Spec.Version == ambassadorChartVersion && meta.GetResourceCondition(hr, meta.ReadyCondition).Reason == meta2.ReconciliationSucceededReason
		}, defaultTimeout, time.Second).Should(BeTrue())

		By("Ensuring that the applications rollback to their starting version")
		Eventually(func() bool {
			hr := &fluxhelmv2beta1.HelmRelease{}
			hrKey := types.NamespacedName{Name: ambassador, Namespace: name}
			if err := k8sClient.Get(ctx, hrKey, hr); err != nil {
				return false
			}
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return hr.Spec.Chart.Spec.Version == ambassadorOldChartVersion && meta.GetResourceCondition(hr, meta.ReadyCondition).Reason == meta2.ReconciliationSucceededReason && appGroup.GetReadyCondition() == meta.WorkflowFailedReason && appGroup.GetWorkflowCondition(v1alpha1.Rollback) == meta.SucceededReason
		}, defaultTimeout, time.Second).Should(BeTrue())
	})

	It("Should create the bookinfo and then delete it while in progress", func() {
		By("Applying the bookinfo object to the cluster")
		err := k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Making sure that the workflow goes into a running state")
		Eventually(func() bool {
			wf := &v1alpha13.Workflow{}
			wfKey := types.NamespacedName{Name: name, Namespace: defaultNamespace}
			_ = k8sClient.Get(ctx, wfKey, wf)
			return string(wf.Status.Phase) == string(v1alpha13.NodeRunning)
		}, time.Minute, time.Second).Should(BeTrue())

		By("Waiting for the bookinfo object to reach a progressing reason")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.ProgressingReason
		}, time.Minute, time.Second).Should(BeTrue())

		By("Waiting for the ambassador helm release to be ready")
		Eventually(func() bool {
			hr := &fluxhelmv2beta1.HelmRelease{}
			hrKey := types.NamespacedName{Name: ambassador, Namespace: name}
			if err := k8sClient.Get(ctx, hrKey, hr); err != nil {
				return false
			}
			readyCondition := meta.GetResourceCondition(hr, meta.ReadyCondition)
			if readyCondition == nil {
				return false
			}
			return readyCondition.Reason == meta2.ReconciliationSucceededReason
		}, time.Minute*2, time.Second).Should(BeTrue())

		By("Waiting for all the HelmReleases to delete")
		err = k8sClient.Delete(ctx, appGroup, client.PropagationPolicy(metav1.DeletePropagationForeground))
		Expect(err).NotTo(HaveOccurred())

		By("Making sure that the workflow goes into a suspended state")
		Eventually(func() bool {
			wf := &v1alpha13.Workflow{}
			wfKey := types.NamespacedName{Name: name, Namespace: defaultNamespace}
			_ = k8sClient.Get(ctx, wfKey, wf)
			return wf.Spec.Suspend != nil && *wf.Spec.Suspend
		}, time.Minute, time.Second).Should(BeTrue())

		By("Waiting for the Workflow to delete all the HelmReleases")
		Eventually(func() bool {
			hr := &fluxhelmv2beta1.HelmReleaseList{}
			if err := k8sClient.List(ctx, hr, client.InNamespace(name)); err != nil {
				return false
			}
			return len(hr.Items) == 0
		}, time.Minute*3, time.Second).Should(BeTrue())

		By("Waiting for all the Workflows to be cleaned up from the cluster")
		Eventually(func() bool {
			wfList := &v1alpha13.WorkflowList{}
			if err := k8sClient.List(ctx, wfList, client.InNamespace(name)); err != nil {
				return false
			}
			return len(wfList.Items) == 0
		}, time.Minute, time.Second).Should(BeTrue())
	})

	It("Should delete the application group if reverse workflow is removed", func() {
		appGroup = smallAppGroup(name, defaultNamespace, name)
		key = client.ObjectKeyFromObject(appGroup)

		By("Applying the bookinfo object to the cluster")
		err := k8sClient.Create(ctx, appGroup)
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the bookinfo object to reach a succeeded reason")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, key, appGroup); err != nil {
				return false
			}
			return appGroup.GetReadyCondition() == meta.SucceededReason
		}, defaultTimeout, time.Second).Should(BeTrue())

		By("Deleting the application group and deleting the workflow")
		err = k8sClient.Delete(ctx, appGroup)
		Expect(err).To(BeNil())
		wf := &v1alpha13.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: defaultNamespace,
			},
		}
		err = k8sClient.Delete(ctx, wf)
		Expect(err).To(BeNil())

		Eventually(func() bool {
			return errors.IsNotFound(k8sClient.Get(ctx, key, appGroup))
		}, time.Second*30, time.Second).Should(BeTrue())

		It("should succeed to remove the new helm releases on application rollback", func() {
			applicationGroup := &v1alpha1.ApplicationGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: defaultNamespace,
				},
				Spec: v1alpha1.ApplicationGroupSpec{
					Applications: []v1alpha1.Application{
						podinfoApplication(name),
					},
				},
			}

			key := client.ObjectKeyFromObject(applicationGroup)

			By("Applying the bookinfo object to the cluster")
			err := k8sClient.Create(ctx, applicationGroup)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the bookinfo object to reach a succeeded reason")
			Eventually(func() bool {
				applicationGroup = &v1alpha1.ApplicationGroup{}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				_, exist := applicationGroup.Annotations[v1alpha1.LastSuccessfulAnnotation]
				return applicationGroup.GetReadyCondition() == meta.SucceededReason && exist

			}, defaultTimeout, time.Second).Should(BeTrue())

			By("adding applications to the application group and intentionally timing out the last application")
			patch := client.MergeFrom(applicationGroup.DeepCopy())
			applicationGroup.Spec.Applications = append(applicationGroup.Spec.Applications, ambassadorApplication(name, podinfo), bookinfoApplication(name, ambassador))
			applicationGroup.Spec.Applications[2].Spec.Release.Timeout = &metav1.Duration{Duration: time.Second}

			err = k8sClient.Patch(ctx, applicationGroup, patch)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the application group to start its upgrade")
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.ProgressingReason && applicationGroup.Generation > 1
			}, time.Second*30, time.Second).Should(BeTrue())

			By("waiting for newer applications to rollout")
			Eventually(func() bool {
				helmReleases := &fluxhelmv2beta1.HelmReleaseList{}
				if err := k8sClient.List(ctx, helmReleases, client.InNamespace(name)); err != nil {
					return false
				}
				return len(helmReleases.Items) > 1
			}, defaultTimeout, time.Second).Should(BeTrue())

			By("ensuring that the newer applications are removed")
			Eventually(func() bool {
				helmReleases := &fluxhelmv2beta1.HelmReleaseList{}
				if err := k8sClient.List(ctx, helmReleases, client.InNamespace(name)); err != nil {
					return false
				}
				if err := k8sClient.Get(ctx, key, applicationGroup); err != nil {
					return false
				}
				return applicationGroup.GetReadyCondition() == meta.WorkflowFailedReason && len(helmReleases.Items) == 1
			}, defaultTimeout, time.Second).Should(BeTrue())
		})
	})
})
