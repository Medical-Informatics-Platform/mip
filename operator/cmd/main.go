package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	notebooksv1alpha1 "github.com/Medical-Informatics-Platform/mip/operator/api/v1alpha1"
	"github.com/Medical-Informatics-Platform/mip/operator/internal/controller"
	"github.com/Medical-Informatics-Platform/mip/operator/internal/spawnapi"
)

func main() {
	var mode, probeAddr string
	var leaderElect bool
	var spawnAPIAddr, tlsCertDir, metricsAddr string
	var rateLimit float64
	var rateBurst int
	flag.StringVar(&mode, "mode", "controller", "controller (reconcile Notebooks) or spawn-api (the hubs' HTTPS API).")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&leaderElect, "leader-elect", false, "Enable leader election (a Lease in the pod namespace).")
	flag.StringVar(&spawnAPIAddr, "spawn-api-bind-address", ":8443", "spawn-api: the address the HTTPS API binds to.")
	flag.StringVar(&tlsCertDir, "tls-cert-dir", "/etc/mip-notebook-spawn-api/tls",
		"spawn-api: directory with tls.crt and tls.key, reloaded when they change.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "spawn-api: the address the metrics endpoint binds to; 0 disables it.")
	flag.Float64Var(&rateLimit, "spawn-api-rate-limit", 5, "spawn-api: requests per second per namespace.")
	flag.IntVar(&rateBurst, "spawn-api-rate-burst", 20, "spawn-api: request burst per namespace.")
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog := ctrl.Log.WithName("setup")

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(notebooksv1alpha1.AddToScheme(scheme))

	switch mode {
	case "controller":
		runController(setupLog, scheme, probeAddr, leaderElect)
	case "spawn-api":
		runSpawnAPI(setupLog, scheme, probeAddr, metricsAddr, spawnAPIAddr, tlsCertDir, rate.Limit(rateLimit), rateBurst)
	default:
		setupLog.Error(nil, "--mode must be controller or spawn-api", "mode", mode)
		os.Exit(1)
	}
}

func runController(setupLog logr.Logger, scheme *runtime.Scheme, probeAddr string, leaderElect bool) {
	// The operator is bound per namespace (RoleBinding to its ClusterRole), so
	// its cache must be limited to exactly those namespaces.
	watchNamespaces := map[string]cache.Config{}
	for _, ns := range strings.Split(os.Getenv("WATCH_NAMESPACES"), ",") {
		if ns = strings.TrimSpace(ns); ns != "" {
			watchNamespaces[ns] = cache.Config{}
		}
	}
	if len(watchNamespaces) == 0 {
		setupLog.Error(nil, "WATCH_NAMESPACES must list at least one namespace")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Cache:                  cache.Options{DefaultNamespaces: watchNamespaces},
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         leaderElect,
		LeaderElectionID:       "mip-notebook-operator.mip.ebrains.eu",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := (&controller.NotebookReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		ExamplesClaim: os.Getenv("EXAMPLES_CLAIM"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Notebook")
		os.Exit(1)
	}
	utilruntime.Must(mgr.AddHealthzCheck("healthz", healthz.Ping))
	utilruntime.Must(mgr.AddReadyzCheck("readyz", healthz.Ping))

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// runSpawnAPI serves the hubs' API under its own ServiceAccount: Notebooks
// get/create/delete and Secrets create in the federations it is bound in,
// tokenreviews create. Stateless, so no leader election, and no cache: the
// account may not list or watch.
func runSpawnAPI(setupLog logr.Logger, scheme *runtime.Scheme, probeAddr, metricsAddr, addr, certDir string,
	limit rate.Limit, burst int) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	c, err := client.New(mgr.GetConfig(), client.Options{Scheme: scheme, Mapper: mgr.GetRESTMapper()})
	if err != nil {
		setupLog.Error(err, "unable to create client")
		os.Exit(1)
	}
	certs, err := certwatcher.New(filepath.Join(certDir, "tls.crt"), filepath.Join(certDir, "tls.key"))
	if err != nil {
		setupLog.Error(err, "unable to load the TLS certificate", "dir", certDir)
		os.Exit(1)
	}
	utilruntime.Must(mgr.Add(certs))

	auth := &spawnapi.TokenReviewAuthenticator{Client: c}
	srv := spawnapi.NewServer(c, auth, limit, burst, ctrl.Log.WithName("spawn-api"))
	listener := &spawnapi.Listener{Addr: addr, Handler: srv.Handler(), GetCertificate: certs.GetCertificate}
	utilruntime.Must(mgr.Add(listener))
	utilruntime.Must(mgr.AddHealthzCheck("healthz", healthz.Ping))
	utilruntime.Must(mgr.AddReadyzCheck("readyz", listener.ReadyCheck))

	setupLog.Info("starting spawn API", "address", addr)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running spawn API")
		os.Exit(1)
	}
}
