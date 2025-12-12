/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Free Trial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Free-Trial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	stash_cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_cs_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/apimachinery/pkg/restic"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	license "go.bytebuilders.dev/license-verifier/kubernetes"
	"gomodules.xyz/flags"
	"gomodules.xyz/go-sh"
	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	appcatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	v1 "kmodules.xyz/offshoot-api/api/v1"
	"kubedb.dev/apimachinery/apis/config/v1alpha1"
)

func NewCmdRestore() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = mongoOptions{
			waitTimeout: 300,
			setupOptions: restic.SetupOptions{
				ScratchDir:  restic.DefaultScratchDir,
				EnableCache: false,
			},
			defaultDumpOptions: restic.DumpOptions{
				Host:     restic.DefaultHost,
				FileName: MongoDumpFile,
			},
		}
	)

	cmd := &cobra.Command{
		Use:               "restore-mongo",
		Short:             "Restores MongoDB Backup",
		DisableAutoGenTag: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkCommandExists()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "appbinding", "provider", "storage-secret-name", "storage-secret-namespace")

			// prepare client
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				return err
			}
			opt.config = config

			opt.stashClient, err = stash_cs.NewForConfig(config)
			if err != nil {
				return err
			}
			opt.kubeClient, err = kubernetes.NewForConfig(config)
			if err != nil {
				return err
			}
			opt.catalogClient, err = appcatalog_cs.NewForConfig(config)
			if err != nil {
				return err
			}

			targetRef := api_v1beta1.TargetRef{
				APIVersion: appcatalog.SchemeGroupVersion.String(),
				Kind:       appcatalog.ResourceKindApp,
				Name:       opt.appBindingName,
				Namespace:  opt.appBindingNamespace,
			}
			var restoreOutput *restic.RestoreOutput
			restoreOutput, err = opt.restoreMongoDB(targetRef)
			if err != nil {
				restoreOutput = &restic.RestoreOutput{
					RestoreTargetStatus: api_v1beta1.RestoreMemberStatus{
						Ref: targetRef,
						Stats: []api_v1beta1.HostRestoreStats{
							{
								Hostname: opt.defaultDumpOptions.Host,
								Phase:    api_v1beta1.HostRestoreFailed,
								Error:    err.Error(),
							},
						},
					},
				}
			}
			// If output directory specified, then write the output in "output.json" file in the specified directory
			if opt.outputDir != "" {
				return restoreOutput.WriteOutput(filepath.Join(opt.outputDir, restic.DefaultOutputFileName))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opt.mongoArgs, "mongo-args", opt.mongoArgs, "Additional arguments")
	cmd.Flags().Int32Var(&opt.waitTimeout, "wait-timeout", opt.waitTimeout, "Number of seconds to wait for the database to be ready")

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.namespace, "namespace", "default", "Namespace of Backup/Restore Session")
	cmd.Flags().StringVar(&opt.appBindingName, "appbinding", opt.appBindingName, "Name of the app binding")
	cmd.Flags().StringVar(&opt.appBindingNamespace, "appbinding-namespace", opt.appBindingNamespace, "Namespace of the app binding")
	cmd.Flags().StringVar(&opt.restoreSessionName, "restoresession", opt.restoreSessionName, "Name of the respective RestoreSession object")
	cmd.Flags().IntVar(&opt.maxConcurrency, "max-concurrency", 3, "maximum concurrent backup process to run to take backup from each replicasets")

	cmd.Flags().StringVar(&opt.setupOptions.Provider, "provider", opt.setupOptions.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&opt.setupOptions.Bucket, "bucket", opt.setupOptions.Bucket, "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&opt.setupOptions.Endpoint, "endpoint", opt.setupOptions.Endpoint, "Endpoint for s3/s3 compatible backend or REST server URL")
	cmd.Flags().BoolVar(&opt.setupOptions.InsecureTLS, "insecure-tls", opt.setupOptions.InsecureTLS, "InsecureTLS for TLS secure s3/s3 compatible backend")
	cmd.Flags().StringVar(&opt.setupOptions.Region, "region", opt.setupOptions.Region, "Region for s3/s3 compatible backend")
	cmd.Flags().StringVar(&opt.setupOptions.Path, "path", opt.setupOptions.Path, "Directory inside the bucket where backup will be stored")
	cmd.Flags().StringVar(&opt.setupOptions.ScratchDir, "scratch-dir", opt.setupOptions.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&opt.setupOptions.EnableCache, "enable-cache", opt.setupOptions.EnableCache, "Specify whether to enable caching for restic")
	cmd.Flags().Int64Var(&opt.setupOptions.MaxConnections, "max-connections", opt.setupOptions.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")

	cmd.Flags().StringVar(&opt.storageSecret.Name, "storage-secret-name", opt.storageSecret.Name, "Name of the storage secret")
	cmd.Flags().StringVar(&opt.storageSecret.Namespace, "storage-secret-namespace", opt.storageSecret.Namespace, "Namespace of the storage secret")
	cmd.Flags().StringVar(&opt.authenticationDatabase, "authentication-database", "admin", "Specify the authentication database")

	cmd.Flags().StringVar(&opt.defaultDumpOptions.Host, "hostname", opt.defaultDumpOptions.Host, "Name of the host machine")
	cmd.Flags().StringVar(&opt.defaultDumpOptions.SourceHost, "source-hostname", opt.defaultDumpOptions.SourceHost, "Name of the host whose data will be restored")
	cmd.Flags().StringVar(&opt.defaultDumpOptions.Snapshot, "snapshot", opt.defaultDumpOptions.Snapshot, "Snapshot to dump")

	cmd.Flags().StringVar(&opt.outputDir, "output-dir", opt.outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	return cmd
}

func (opt *mongoOptions) restoreMongoDB(targetRef api_v1beta1.TargetRef) (*restic.RestoreOutput, error) {
	var err error
	err = license.CheckLicenseEndpoint(opt.config, licenseApiService, SupportedProducts)
	if err != nil {
		return nil, err
	}

	opt.setupOptions.StorageSecret, err = opt.kubeClient.CoreV1().Secrets(opt.storageSecret.Namespace).Get(context.TODO(), opt.storageSecret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// apply nice, ionice settings from env
	opt.setupOptions.Nice, err = v1.NiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	opt.setupOptions.IONice, err = v1.IONiceSettingsFromEnv()
	if err != nil {
		return nil, err
	}

	appBinding, err := opt.catalogClient.AppcatalogV1alpha1().AppBindings(opt.appBindingNamespace).Get(context.TODO(), opt.appBindingName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	authSecret, err := opt.kubeClient.CoreV1().Secrets(opt.appBindingNamespace).Get(context.TODO(), appBinding.Spec.Secret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = appBinding.TransformSecret(opt.kubeClient, authSecret.Data)
	if err != nil {
		return nil, err
	}

	var tlsSecret *core.Secret
	if appBinding.Spec.TLSSecret != nil {
		tlsSecret, err = opt.kubeClient.CoreV1().Secrets(opt.appBindingNamespace).Get(context.TODO(), appBinding.Spec.TLSSecret.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	hostname, err := appBinding.Hostname()
	if err != nil {
		return nil, err
	}

	var isSrv bool
	port := int32(27017)
	if appBinding.Spec.ClientConfig.URL != nil {
		isSrv, err = isSrvConnection(*appBinding.Spec.ClientConfig.URL)
		if err != nil {
			return nil, err
		}
	}

	// Checked for Altlas and DigitalOcean srv format connection string don't give port.
	// mongodump --uri format not support port.

	if !isSrv {
		port, err = appBinding.Port()
		if err != nil {
			return nil, err
		}
	}

	// unmarshal parameter is the field has value
	parameters := v1alpha1.MongoDBConfiguration{}
	if appBinding.Spec.Parameters != nil {
		if err = json.Unmarshal(appBinding.Spec.Parameters.Raw, &parameters); err != nil {
			klog.Errorf("unable to unmarshal appBinding.Spec.Parameters.Raw. Reason: %v", err)
		}
	}

	// Stash operator does not know how many hosts this plugin will restore. It sets totalHosts field of respective RestoreSession to 1.
	// We must update the totalHosts field to the actual number of hosts it will restore.
	// Otherwise, RestoreSession will stuck in "Running" state.
	// Total hosts for MongoDB:
	// 1. For stand-alone MongoDB, totalHosts=1.
	// 2. For MongoDB ReplicaSet, totalHosts=1.
	// 3. For sharded MongoDB, totalHosts=(number of shard + 1) // extra 1 for config server
	// So, for stand-alone MongoDB and MongoDB ReplicaSet, we don't have to do anything.
	// We only need to update totalHosts field for sharded MongoDB

	restoreSession, err := opt.stashClient.StashV1beta1().RestoreSessions(opt.namespace).Get(context.TODO(), opt.restoreSessionName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	opt.totalHosts = 1
	// For sharded MongoDB, parameter.ConfigServer will not be empty
	if parameters.ConfigServer != "" {
		opt.totalHosts = len(parameters.ReplicaSets) + 1 // for each shard there will be one key in parameters.ReplicaSet
		_, err = stash_cs_util.UpdateRestoreSessionStatus(
			context.TODO(),
			opt.stashClient.StashV1beta1(),
			restoreSession.ObjectMeta,
			func(status *api_v1beta1.RestoreSessionStatus) (types.UID, *api_v1beta1.RestoreSessionStatus) {
				status.TotalHosts = pointer.Int32P(int32(len(parameters.ReplicaSets) + 1)) // for each shard there will be one key in parameters.ReplicaSet
				return restoreSession.UID, status
			},
			metav1.UpdateOptions{},
		)
		if err != nil {
			return nil, err
		}
	}
	var tlsEnable bool
	if appBinding.Spec.ClientConfig.CABundle != nil {
		tlsEnable = true
	}

	if tlsEnable {
		if err := os.WriteFile(filepath.Join(opt.setupOptions.ScratchDir, MongoTLSCertFileName), appBinding.Spec.ClientConfig.CABundle, os.ModePerm); err != nil {
			return nil, errors.Wrap(err, "failed to write key for CA certificate")
		}
		mongoCreds = []any{
			"--tls",
			"--tlsCAFile", filepath.Join(opt.setupOptions.ScratchDir, MongoTLSCertFileName),
			"--tlsCertificateKeyFile", filepath.Join(opt.setupOptions.ScratchDir, MongoClientPemFileName),
		}
		dumpCreds = []any{
			"--ssl",
			fmt.Sprintf("--sslCAFile=%s", filepath.Join(opt.setupOptions.ScratchDir, MongoTLSCertFileName)),
			fmt.Sprintf("--sslPEMKeyFile=%s", filepath.Join(opt.setupOptions.ScratchDir, MongoClientPemFileName)),
		}

		// get certificate secret to get client certificate
		var pemBytes []byte
		var ok bool
		pemBytes, ok = tlsSecret.Data[MongoClientPemFileName]
		if !ok {
			crt, ok := tlsSecret.Data[core.TLSCertKey]
			if !ok {
				return nil, errors.Wrap(err, "unable to retrieve tls.crt from secret.")
			}
			key, ok := tlsSecret.Data[core.TLSPrivateKeyKey]
			if !ok {
				return nil, errors.Wrap(err, "unable to retrieve tls.key from secret.")
			}
			pemBytes = append(crt[:], []byte("\n")...)
			pemBytes = append(pemBytes, key...)
		}
		if err := os.WriteFile(filepath.Join(opt.setupOptions.ScratchDir, MongoClientPemFileName), pemBytes, os.ModePerm); err != nil {
			return nil, errors.Wrap(err, "failed to write client certificate")
		}
		user, err := getSSLUser(filepath.Join(opt.setupOptions.ScratchDir, MongoClientPemFileName))
		if err != nil {
			return nil, errors.Wrap(err, "unable to get user from ssl.")
		}
		userAuth := []any{
			fmt.Sprintf("--username=%s", user),
			"--authenticationMechanism=MONGODB-X509",
			"--authenticationDatabase=$external",
		}
		mongoCreds = append(mongoCreds, userAuth...)
		dumpCreds = append(dumpCreds, userAuth...)

	} else {
		userAuth := []any{
			fmt.Sprintf("--username=%s", authSecret.Data[MongoUserKey]),
			fmt.Sprintf("--password=%s", authSecret.Data[MongoPasswordKey]),
			fmt.Sprintf("--authenticationDatabase=%s", opt.authenticationDatabase),
		}
		mongoCreds = append(mongoCreds, userAuth...)
		dumpCreds = append(dumpCreds, userAuth...)
	}

	getDumpOpts := func(mongoDSN, hostKey string, isStandalone bool) restic.DumpOptions {
		klog.Infoln("processing backupOptions for ", mongoDSN)
		dumpOpt := restic.DumpOptions{
			Host:       hostKey,
			SourceHost: hostKey,
			FileName:   opt.defaultDumpOptions.FileName,
			Snapshot:   opt.getSnapshotForHost(hostKey, restoreSession.Spec.Target.Rules),
		}

		uri := opt.buildMongoURI(mongoDSN, port, isStandalone, isSrv, tlsEnable)

		// setup pipe command
		restoreCmd := restic.Command{
			Name: MongoRestoreCMD,
			Args: []any{
				"--uri", fmt.Sprintf("\"%s\"", uri),
				"--archive",
			},
		}
		if tlsEnable {
			restoreCmd.Args = append(restoreCmd.Args,
				fmt.Sprintf("--sslCAFile=%s", getOptionValue(dumpCreds, "--sslCAFile")),
				fmt.Sprintf("--sslPEMKeyFile=%s", getOptionValue(dumpCreds, "--sslPEMKeyFile")))
		}

		userArgs := strings.Fields(opt.mongoArgs)

		if !isStandalone {
			// - port is already added in mongoDSN with replicasetName/host:port format.
			// - oplog is enabled automatically for replicasets.
			// Don't use --oplogReplay if user specify any of these arguments through opt.mongoArgs
			// 1. --db
			// 2. --collection
			// 3. --nsInclude
			// 4. --nsExclude
			// xref: https://docs.mongodb.com/manual/reference/program/mongorestore/#cmdoption-mongorestore-oplogreplay
			forbiddenArgs := sets.New[string](
				"-d", "--db",
				"-c", "--collection",
				"--nsInclude",
				"--nsExclude",
			)
			if !containsArg(userArgs, forbiddenArgs) {
				restoreCmd.Args = append(restoreCmd.Args, "--oplogReplay")
			}
		}

		for _, arg := range userArgs {
			// illegal argument combination: cannot specify --db and --uri
			if !strings.Contains(arg, "--db") {
				restoreCmd.Args = append(restoreCmd.Args, arg)
			}
		}

		// add the restore command to the pipeline
		dumpOpt.StdoutPipeCommands = append(dumpOpt.StdoutPipeCommands, restoreCmd)
		return dumpOpt
	}

	// set opt.maxConcurrency
	if len(parameters.ReplicaSets) <= 1 {
		opt.maxConcurrency = 1
	}

	// If parameters.ReplicaSets is not empty, then replicaset hosts are given in key:value pair,
	// where, keys are host-0,host-1 etc and values are the replicaset dsn of one replicaset component
	//
	// Procedure of restore in a sharded or replicaset cluster
	// - Restore the CSRS primary mongod data files.
	// - Restore Each Shard Replica Set
	// ref: https://docs.mongodb.com/manual/tutorial/backup-sharded-cluster-with-database-dumps/

	if parameters.ConfigServer != "" {
		opt.dumpOptions = append(opt.dumpOptions, getDumpOpts(extractHost(parameters.ConfigServer), MongoConfigSVRHostKey, false))
	}

	for key, host := range parameters.ReplicaSets {
		opt.dumpOptions = append(opt.dumpOptions, getDumpOpts(extractHost(host), key, false))
	}

	// if parameters.ReplicaSets is nil, then perform normal backup with clientconfig.Service.Name mongo dsn
	if parameters.ReplicaSets == nil {
		opt.dumpOptions = append(opt.dumpOptions, getDumpOpts(hostname, restic.DefaultHost, true))
	}

	klog.Infoln("processing restore.")

	waitForDBReady(hostname, port, opt.waitTimeout)

	resticWrapper, err := restic.NewResticWrapper(opt.setupOptions)
	if err != nil {
		return nil, err
	}
	// hide password, don't print cmd
	resticWrapper.HideCMD()

	// Run dump
	restoreOutput, err := resticWrapper.ParallelDump(opt.dumpOptions, targetRef, opt.maxConcurrency)
	if err != nil {
		return nil, err
	}

	if parameters.ConfigServer != "" {
		err = dropTempReshardCollection(parameters.ConfigServer)
		if err != nil {
			klog.Errorf("error while deleting temporary reshard collection for %v. error: %v", parameters.ConfigServer, err)
			return nil, err
		}
	}

	return restoreOutput, nil
}

func dropTempReshardCollection(configsvrDSN string) error {
	args := append([]any{
		"config",
		"--host", configsvrDSN,
		"--quiet",
		"--eval", `db.reshardingOperations_temp.drop()`,
	}, mongoCreds...)

	return sh.Command(MongoCMD, args...).Command("/usr/bin/tail", "-1").Run()
}

func (opt *mongoOptions) getSnapshotForHost(hostname string, rules []api_v1beta1.Rule) string {
	var hostSnapshot string
	for _, rule := range rules {
		if len(rule.TargetHosts) == 0 || containsString(rule.TargetHosts, hostname) {
			hostSnapshot = rule.Snapshots[0]
			// if rule has empty targetHost then check further rules to see if any other rule with non-empty targetHost matches
			if len(rule.TargetHosts) == 0 {
				continue
			} else {
				return hostSnapshot
			}
		}
	}
	return hostSnapshot
}
