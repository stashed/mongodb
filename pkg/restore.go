package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	"kubedb.dev/apimachinery/apis/config/v1alpha1"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"
)

func NewCmdRestore() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		namespace      string
		appBindingName string
		outputDir      string
		mongoArgs      string
		maxConcurrency int

		dumpOpts []restic.DumpOptions
		setupOpt = restic.SetupOptions{
			ScratchDir:  restic.DefaultScratchDir,
			EnableCache: false,
		}
		defaultDumpOpt = restic.DumpOptions{
			Host:     restic.DefaultHost,
			FileName: MongoDumpFile,
		}
		metrics = restic.MetricsOptions{
			JobName: JobMongoBackup,
		}
	)

	cmd := &cobra.Command{
		Use:               "restore-mongo",
		Short:             "Restores Mongo DB Backup",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "app-binding", "provider", "secret-dir")

			// apply nice, ionice settings from env
			var err error
			setupOpt.Nice, err = util.NiceSettingsFromEnv()
			if err != nil {
				return util.HandleResticError(outputDir, restic.DefaultOutputFileName, err)
			}
			setupOpt.IONice, err = util.IONiceSettingsFromEnv()
			if err != nil {
				return util.HandleResticError(outputDir, restic.DefaultOutputFileName, err)
			}

			// prepare client
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				return err
			}
			kubeClient, err := kubernetes.NewForConfig(config)
			if err != nil {
				return err
			}
			appCatalogClient, err := appcatalog_cs.NewForConfig(config)
			if err != nil {
				return err
			}

			// get app binding
			appBinding, err := appCatalogClient.AppcatalogV1alpha1().AppBindings(namespace).Get(appBindingName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// get secret
			appBindingSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(appBinding.Spec.Secret.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			MongoDBRootUser = string(appBindingSecret.Data[MongoUserKey])
			MongoDBRootPassword = string(appBindingSecret.Data[MongoPasswordKey])

			// unmarshal parameter is the field has value
			parameters := v1alpha1.MongoDBConfiguration{}
			if appBinding.Spec.Parameters != nil {
				if err = json.Unmarshal(appBinding.Spec.Parameters.Raw, &parameters); err != nil {
					log.Errorf("unable to unmarshal appBinding.Spec.Parameters.Raw. Reason: %v", err)
				}
			}

			if appBinding.Spec.ClientConfig.CABundle != nil {
				if err := ioutil.WriteFile(filepath.Join(setupOpt.ScratchDir, MongoCACertFile), appBinding.Spec.ClientConfig.CABundle, os.ModePerm); err != nil {
					return errors.Wrap(err, "failed to write key for CA certificate")
				}
				tlsArgs = fmt.Sprintf("--ssl --sslCAFile=%v", filepath.Join(setupOpt.ScratchDir, MongoCACertFile))

				if parameters.CertificateSecret != "" {
					// get certificate secret to get client certificate
					certificateSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(parameters.CertificateSecret, metav1.GetOptions{})
					if err != nil {
						return err
					}
					if err := ioutil.WriteFile(filepath.Join(setupOpt.ScratchDir, MongoClientCertFile), certificateSecret.Data[parameters.ClientCertKey], os.ModePerm); err != nil {
						return errors.Wrap(err, "failed to write client certificate")
					}
					tlsArgs += fmt.Sprintf(" --sslPEMKeyFile=%v", filepath.Join(setupOpt.ScratchDir, MongoClientCertFile))
				}
			}

			getDumpOpts := func(mongoDSN, hostKey string, isStandalone bool) restic.DumpOptions {
				log.Infoln("processing backupOptions for ", mongoDSN)
				dumpOpt := restic.DumpOptions{
					Host:     hostKey,
					FileName: defaultDumpOpt.FileName,
					Snapshot: defaultDumpOpt.Snapshot,
				}

				// setup pipe command
				dumpOpt.StdoutPipeCommand = restic.Command{
					Name: MongoRestoreCMD,
					Args: []interface{}{
						"--host=" + mongoDSN,
						"--username=" + string(appBindingSecret.Data[MongoUserKey]),
						"--password=" + string(appBindingSecret.Data[MongoPasswordKey]),
						"--archive",
						tlsArgs,
						mongoArgs,
					},
				}
				if isStandalone {
					dumpOpt.StdoutPipeCommand.Args = append(dumpOpt.StdoutPipeCommand.Args, "--port="+fmt.Sprint(appBinding.Spec.ClientConfig.Service.Port))
				} else {
					// - port is already added in mongoDSN with replicasetName/host:port format.
					// - oplog is enabled automatically for replicasets.
					dumpOpt.StdoutPipeCommand.Args = append(dumpOpt.StdoutPipeCommand.Args, "--oplogReplay")
				}
				return dumpOpt
			}

			// set maxConcurrency
			if len(parameters.ReplicaSets) <= 1 {
				maxConcurrency = 1
			}

			// If parameters.ReplicaSets is not empty, then replicaset hosts are given in key:value pair,
			// where, keys are host-0,host-1 etc and values are the replicaset dsn of one replicaset component
			//
			// Procedure of restore in a sharded or replicaset cluster
			// - Restore the CSRS primary mongod data files.
			// - Restore Each Shard Replica Set
			// ref: https://docs.mongodb.com/manual/tutorial/backup-sharded-cluster-with-database-dumps/

			if parameters.ConfigServer != "" {
				dumpOpts = append(dumpOpts, getDumpOpts(parameters.ConfigServer, MongoConfigSVRHostKey, false))
			}

			for key, host := range parameters.ReplicaSets {
				dumpOpts = append(dumpOpts, getDumpOpts(host, key, false))
			}

			// if parameters.ReplicaSets is nil, then perform normal backup with clientconfig.Service.Name mongo dsn
			if parameters.ReplicaSets == nil {
				dumpOpts = append(dumpOpts, getDumpOpts(appBinding.Spec.ClientConfig.Service.Name, restic.DefaultHost, true))
			}

			log.Infoln("processing restore.")

			// init restic wrapper
			resticWrapper, err := restic.NewResticWrapper(setupOpt)
			if err != nil {
				return err
			}
			// hide password, don't print cmd
			resticWrapper.HideCMD()

			// Run dump
			dumpOutput, backupErr := resticWrapper.ParallelDump(dumpOpts, maxConcurrency)
			// If metrics are enabled then generate metrics
			if metrics.Enabled {
				err := dumpOutput.HandleMetrics(&metrics, backupErr)
				if err != nil {
					return kerrors.NewAggregate([]error{backupErr, err})
				}
			}
			// If output directory specified, then write the output in "output.json" file in the specified directory
			if backupErr == nil && outputDir != "" {
				err := dumpOutput.WriteOutput(filepath.Join(outputDir, restic.DefaultOutputFileName))
				if err != nil {
					return err
				}
			}
			return backupErr
		},
	}

	cmd.Flags().StringVar(&mongoArgs, "mongo-args", mongoArgs, "Additional arguments")

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "Namespace of Backup/Restore Session")
	cmd.Flags().StringVar(&appBindingName, "app-binding", appBindingName, "Name of the app binding")
	cmd.Flags().IntVar(&maxConcurrency, "max-concurrency", 3, "maximum concurrent backup process to run to take backup from each replicasets")

	cmd.Flags().StringVar(&setupOpt.Provider, "provider", setupOpt.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&setupOpt.Bucket, "bucket", setupOpt.Bucket, "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&setupOpt.Endpoint, "endpoint", setupOpt.Endpoint, "Endpoint for s3/s3 compatible backend")
	cmd.Flags().StringVar(&setupOpt.URL, "rest-server-url", setupOpt.URL, "URL for rest backend")
	cmd.Flags().StringVar(&setupOpt.Path, "path", setupOpt.Path, "Directory inside the bucket where backup will be stored")
	cmd.Flags().StringVar(&setupOpt.SecretDir, "secret-dir", setupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().StringVar(&setupOpt.ScratchDir, "scratch-dir", setupOpt.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&setupOpt.EnableCache, "enable-cache", setupOpt.EnableCache, "Specify whether to enable caching for restic")
	cmd.Flags().IntVar(&setupOpt.MaxConnections, "max-connections", setupOpt.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")

	cmd.Flags().StringVar(&defaultDumpOpt.Host, "hostname", defaultDumpOpt.Host, "Name of the host machine")
	cmd.Flags().StringVar(&defaultDumpOpt.Snapshot, "snapshot", defaultDumpOpt.Snapshot, "Snapshot to dump")

	cmd.Flags().StringVar(&outputDir, "output-dir", outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	cmd.Flags().BoolVar(&metrics.Enabled, "metrics-enabled", metrics.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&metrics.PushgatewayURL, "metrics-pushgateway-url", metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringVar(&metrics.MetricFileDir, "metrics-dir", metrics.MetricFileDir, "Directory where to write metric.prom file (keep empty if you don't want to write metric in a text file)")
	cmd.Flags().StringSliceVar(&metrics.Labels, "metrics-labels", metrics.Labels, "Labels to apply in exported metrics")

	return cmd
}
