// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/Azure/Orkestra/pkg/plugins"
	"github.com/Azure/Orkestra/pkg/utils"
	"github.com/Azure/Orkestra/pkg/workflow"

	"github.com/Azure/Orkestra/pkg/registry"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	"github.com/Azure/Orkestra/controllers"
	v1alpha13 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fluxhelmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	// +kubebuilder:scaffold:imports
)

const (
	stagingRepoURLEnv = "STAGING_REPO_URL"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = orkestrav1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme

	// Add Argo Workflow scheme to operator
	_ = v1alpha13.AddToScheme(scheme)

	// Add HelmRelease scheme to operator
	_ = fluxhelmv2beta1.AddToScheme(scheme)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var configPath string
	var stagingRepoURL string
	var tempChartStoreTargetDir string
	var disableRemediation bool
	var cleanupDownloadedCharts bool
	var debug bool
	var workflowParallelism int64
	var logLevel int
	var withPlugins string

	flag.StringVar(&metricsAddr, "metrics-addr", ":8081", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&configPath, "config", "", "The path to the controller config file")
	flag.StringVar(&stagingRepoURL, "staging-repo-url", "", "The URL for the helm registry used for staging artifacts (ENV - STAGING_REPO_URL). NOTE: Flag overrides env value")
	flag.StringVar(&tempChartStoreTargetDir, "chart-store-path", "", "The temporary storage path for the downloaded and staged chart artifacts")
	flag.BoolVar(&disableRemediation, "disable-remediation", false, "Disable the remediation (delete/rollback) of the workflow on failure (useful if you wish to debug failures in the workflow/executor container")
	flag.BoolVar(&cleanupDownloadedCharts, "cleanup-downloaded-charts", false, "Enable/disable the cleanup of the charts downloaded to the chart-store-path")
	flag.BoolVar(&debug, "debug", false, "Enable debug run of the appgroup controller")
	flag.Int64Var(&workflowParallelism, "workflow-parallelism", 10, "Specifies the max number of workflow pods that can be executed in parallel")
	flag.IntVar(&logLevel, "log-level", 0, "Log Level")
	// plugin related flags
	flag.StringVar(&withPlugins, "with-plugins", "", "Enable/disable specific plugins. Plugins must be provided as a comma seperated list")
	flag.Parse()

	if logLevel > 0 {
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	} else {
		ctrl.SetLogger(zap.New(zap.UseDevMode(false)))
	}

	// Start the probe at the very beginning
	probe, err := utils.ProbeHandler(stagingRepoURL, "health")
	if err != nil {
		setupLog.Error(err, "unable to start readiness/liveness probes", "controller", "ApplicationGroup")
		os.Exit(1)
	}

	probe.Start("8086")

	ctrl.Log.V(logLevel)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "fdcf4a0d.azure.microsoft.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Grabbing the values based on the passed helm flags, these values change if we run in debug mode
	stagingHelmURL, workflowHelmURL, tempChartStoreTargetDir := getValues(stagingRepoURL, tempChartStoreTargetDir, debug)

	if stagingHelmURL == "" {
		if s := os.Getenv(stagingRepoURLEnv); s != "" {
			stagingHelmURL = s
		} else {
			setupLog.Error(err, "staging repo URL must be set")
			os.Exit(1)
		}
	}

	rc, err := registry.NewClient(
		ctrl.Log,
		registry.TargetDir(tempChartStoreTargetDir),
	)
	if err != nil {
		setupLog.Error(err, "unable to create new registry client", "controller", "registry-client")
		os.Exit(1)
	}

	// Register the staging helm repository/registry
	// We perform retry on this so that we don't go into a crash loop backoff
	retryChan := make(chan bool)
	retryCtx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	go func() {
		for {
			err = rc.AddRepo(&registry.Config{
				Name: "staging",
				URL:  stagingHelmURL,
			})
			if err != nil {
				setupLog.Info("failed to add staging helm repo, retrying...")
				time.Sleep(time.Second * 5)
			} else {
				retryChan <- true
				break
			}
		}
	}()
	select {
	case <-retryChan:
		cancel()
		close(retryChan)
		setupLog.Info("successfully set-up the local chartmuseum helm repository")
	case <-retryCtx.Done():
		cancel()
		close(retryChan)
		setupLog.Error(err, "pod timed out while trying to setup the helm chart museum...")
		os.Exit(1)
	}

	baseLogger := ctrl.Log.WithName("controllers").WithName("ApplicationGroup")

	// Initialize the plugins
	var enabledPlugins map[string]plugins.Plugin
	if enabledPlugins, err = plugins.DecomposeCSL(withPlugins); err != nil {
		setupLog.Error(err, "unable to decompose plugins")
		os.Exit(1)
	}

	for _, p := range enabledPlugins {
		if err = p.Init(); err != nil {
			setupLog.Error(err, "unable to initialize plugin", "plugin", p.Name())
			os.Exit(1)
		}
	}

	if err = (&controllers.ApplicationGroupReconciler{
		Client:                  mgr.GetClient(),
		Log:                     baseLogger,
		Scheme:                  mgr.GetScheme(),
		RegistryClient:          rc,
		StagingRepoName:         "staging",
		WorkflowClientBuilder:   workflow.NewBuilder(mgr.GetClient(), baseLogger).WithStagingRepo(workflowHelmURL).WithParallelism(workflowParallelism).InNamespace(workflow.GetNamespace()),
		TargetDir:               tempChartStoreTargetDir,
		Recorder:                mgr.GetEventRecorderFor("appgroup-controller"),
		DisableRemediation:      disableRemediation,
		CleanupDownloadedCharts: cleanupDownloadedCharts,
		Plugins:                 enabledPlugins,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApplicationGroup")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// getValues returns the stagingRepoUrl unless the appGroup controller
// is run in a debug mode, then it returns the port forwarded url
func getValues(stagingHelmURL, tempChartStoreTargetDir string, debug bool) (string, string, string) {
	if debug {
		return "http://127.0.0.1:8080", "http://orkestra-chartmuseum.orkestra:8080", os.TempDir()
	}
	return stagingHelmURL, stagingHelmURL, tempChartStoreTargetDir
}
