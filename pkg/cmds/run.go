package cmds

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/appscode/go/log"
	"github.com/appscode/go/runtime"
	stringz "github.com/appscode/go/strings"
	pcm "github.com/coreos/prometheus-operator/pkg/client/monitoring/v1"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	cs "github.com/kubedb/apimachinery/client/typed/kubedb/v1alpha1"
	snapc "github.com/kubedb/apimachinery/pkg/controller/snapshot"
	"github.com/kubedb/apimachinery/pkg/migrator"
	"github.com/kubedb/elasticsearch/pkg/controller"
	"github.com/kubedb/elasticsearch/pkg/docker"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdRun(version string) *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
	)

	opt := controller.Options{
		Docker: docker.Docker{
			Registry:    "kubedb",
			ExporterTag: stringz.Val(version, "canary"),
		},
		OperatorNamespace: namespace(),
		GoverningService:  "kubedb",
		Address:           ":8080",
		AnalyticsClientID: analyticsClientID,
	}

	cmd := &cobra.Command{
		Use:               "run",
		Short:             "Run Elasticsearch in Kubernetes",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get kubernetes config: %s", err)
			}

			client := kubernetes.NewForConfigOrDie(config)
			apiExtKubeClient := crd_cs.NewForConfigOrDie(config)
			extClient := cs.NewForConfigOrDie(config)
			promClient, err := pcm.NewForConfig(config)
			if err != nil {
				log.Fatalln(err)
			}

			cronController := snapc.NewCronController(client, extClient)
			// Start Cron
			cronController.StartCron()
			// Stop Cron
			defer cronController.StopCron()

			tprMigrator := migrator.NewMigrator(client, apiExtKubeClient, extClient)
			err = tprMigrator.RunMigration(
				&api.Elasticsearch{},
				&api.Snapshot{},
				&api.DormantDatabase{},
			)
			if err != nil {
				log.Fatalln(err)
			}

			w := controller.New(client, apiExtKubeClient, extClient, promClient, cronController, opt)
			defer runtime.HandleCrash()

			// Ensuring Custom Resources Definitions
			err = w.Setup()
			if err != nil {
				log.Fatalln(err)
			}

			fmt.Println("Starting operator...")
			w.RunAndHold()
		},
	}

	// operator flags
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.GoverningService, "governing-service", opt.GoverningService, "Governing service for database statefulset")
	cmd.Flags().StringVar(&opt.Docker.Registry, "docker-registry", opt.Docker.Registry, "User provided docker repository")
	cmd.Flags().StringVar(&opt.Docker.ExporterTag, "exporter-tag", opt.Docker.ExporterTag, "Tag of kubedb/operator used as exporter")
	cmd.Flags().StringVar(&opt.Address, "address", opt.Address, "Address to listen on for web interface and telemetry.")

	return cmd
}

func namespace() string {
	if ns := os.Getenv("OPERATOR_NAMESPACE"); ns != "" {
		return ns
	}
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return core.NamespaceDefault
}
