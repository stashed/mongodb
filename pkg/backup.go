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
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	stash_cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stash_cs_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/apimachinery/pkg/restic"
	api_util "stash.appscode.dev/apimachinery/pkg/util"

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

var (
	MongoshCMD   = "/usr/bin/mongosh"
	OpenSSLCMD   = "/usr/bin/openssl"
	mongoCreds   []interface{}
	dumpCreds    []interface{}
	cleanupFuncs []func() error
)

const (
	StashRoleName = "stash-backup"
	StashUserName = "stash-backup"
)

func checkCommandExists() error {
	var err error
	if MongoshCMD, err = exec.LookPath(MongoshCMD); err != nil {
		return fmt.Errorf("unable to look for mongo command. reason: %v", err)
	}
	if OpenSSLCMD, err = exec.LookPath(OpenSSLCMD); err != nil {
		return fmt.Errorf("unable to look for openssl command. reason: %v", err)
	}
	return nil
}

func NewCmdBackup() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = mongoOptions{
			waitTimeout: 300,
			setupOptions: restic.SetupOptions{
				ScratchDir:  restic.DefaultScratchDir,
				EnableCache: false,
			},
			defaultBackupOptions: restic.BackupOptions{
				Host:          restic.DefaultHost,
				StdinFileName: MongoDumpFile,
			},
		}
	)

	cmd := &cobra.Command{
		Use:               "backup-mongo",
		Short:             "Takes a backup of Mongo DB",
		DisableAutoGenTag: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkCommandExists()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer cleanup()

			flags.EnsureRequiredFlags(cmd, "appbinding", "provider", "storage-secret-name", "storage-secret-namespace")

			// catch sigkill signals and gracefully terminate so that cleanup functions are executed.
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				rcvSig := <-sigChan
				cleanup()
				klog.Errorf("Received signal: %s, exiting\n", rcvSig)
				os.Exit(1)
			}()

			// prepare client
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				return err
			}
			opt.config = config

			opt.kubeClient, err = kubernetes.NewForConfig(config)
			if err != nil {
				return err
			}
			opt.stashClient, err = stash_cs.NewForConfig(config)
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
			var backupOutput *restic.BackupOutput
			backupOutput, err = opt.backupMongoDB(targetRef)
			if err != nil {
				backupOutput = &restic.BackupOutput{
					BackupTargetStatus: api_v1beta1.BackupTargetStatus{
						Ref: targetRef,
						Stats: []api_v1beta1.HostBackupStats{
							{
								Hostname: opt.defaultBackupOptions.Host,
								Phase:    api_v1beta1.HostBackupFailed,
								Error:    err.Error(),
							},
						},
					},
				}
			}
			// If output directory specified, then write the output in "output.json" file in the specified directory
			if opt.outputDir != "" {
				return backupOutput.WriteOutput(filepath.Join(opt.outputDir, restic.DefaultOutputFileName))
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
	cmd.Flags().StringVar(&opt.backupSessionName, "backupsession", opt.backupSessionName, "Name of the respective BackupSession object")
	cmd.Flags().IntVar(&opt.maxConcurrency, "max-concurrency", 3, "maximum concurrent backup process to run to take backup from each replicasets")

	cmd.Flags().StringVar(&opt.setupOptions.Provider, "provider", opt.setupOptions.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&opt.setupOptions.Bucket, "bucket", opt.setupOptions.Bucket, "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&opt.setupOptions.Endpoint, "endpoint", opt.setupOptions.Endpoint, "Endpoint for s3/s3 compatible backend or REST server URL")
	cmd.Flags().StringVar(&opt.setupOptions.Region, "region", opt.setupOptions.Region, "Region for s3/s3 compatible backend")
	cmd.Flags().StringVar(&opt.setupOptions.Path, "path", opt.setupOptions.Path, "Directory inside the bucket where backup will be stored")
	cmd.Flags().StringVar(&opt.setupOptions.ScratchDir, "scratch-dir", opt.setupOptions.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&opt.setupOptions.EnableCache, "enable-cache", opt.setupOptions.EnableCache, "Specify whether to enable caching for restic")
	cmd.Flags().Int64Var(&opt.setupOptions.MaxConnections, "max-connections", opt.setupOptions.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")

	cmd.Flags().StringVar(&opt.storageSecret.Name, "storage-secret-name", opt.storageSecret.Name, "Name of the storage secret")
	cmd.Flags().StringVar(&opt.storageSecret.Namespace, "storage-secret-namespace", opt.storageSecret.Namespace, "Namespace of the storage secret")
	cmd.Flags().StringVar(&opt.authenticationDatabase, "authentication-database", "admin", "Specify the authentication database")
	cmd.Flags().StringVar(&opt.defaultBackupOptions.Host, "hostname", opt.defaultBackupOptions.Host, "Name of the host machine")

	cmd.Flags().Int64Var(&opt.defaultBackupOptions.RetentionPolicy.KeepLast, "retention-keep-last", opt.defaultBackupOptions.RetentionPolicy.KeepLast, "Specify value for retention strategy")
	cmd.Flags().Int64Var(&opt.defaultBackupOptions.RetentionPolicy.KeepHourly, "retention-keep-hourly", opt.defaultBackupOptions.RetentionPolicy.KeepHourly, "Specify value for retention strategy")
	cmd.Flags().Int64Var(&opt.defaultBackupOptions.RetentionPolicy.KeepDaily, "retention-keep-daily", opt.defaultBackupOptions.RetentionPolicy.KeepDaily, "Specify value for retention strategy")
	cmd.Flags().Int64Var(&opt.defaultBackupOptions.RetentionPolicy.KeepWeekly, "retention-keep-weekly", opt.defaultBackupOptions.RetentionPolicy.KeepWeekly, "Specify value for retention strategy")
	cmd.Flags().Int64Var(&opt.defaultBackupOptions.RetentionPolicy.KeepMonthly, "retention-keep-monthly", opt.defaultBackupOptions.RetentionPolicy.KeepMonthly, "Specify value for retention strategy")
	cmd.Flags().Int64Var(&opt.defaultBackupOptions.RetentionPolicy.KeepYearly, "retention-keep-yearly", opt.defaultBackupOptions.RetentionPolicy.KeepYearly, "Specify value for retention strategy")
	cmd.Flags().StringSliceVar(&opt.defaultBackupOptions.RetentionPolicy.KeepTags, "retention-keep-tags", opt.defaultBackupOptions.RetentionPolicy.KeepTags, "Specify value for retention strategy")
	cmd.Flags().BoolVar(&opt.defaultBackupOptions.RetentionPolicy.Prune, "retention-prune", opt.defaultBackupOptions.RetentionPolicy.Prune, "Specify whether to prune old snapshot data")
	cmd.Flags().BoolVar(&opt.defaultBackupOptions.RetentionPolicy.DryRun, "retention-dry-run", opt.defaultBackupOptions.RetentionPolicy.DryRun, "Specify whether to test retention policy without deleting actual data")

	cmd.Flags().StringVar(&opt.outputDir, "output-dir", opt.outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	return cmd
}

func (opt *mongoOptions) backupMongoDB(targetRef api_v1beta1.TargetRef) (*restic.BackupOutput, error) {
	var err error
	err = license.CheckLicenseEndpoint(opt.config, licenseApiService, SupportedProducts)
	if err != nil {
		return nil, err
	}

	opt.setupOptions.StorageSecret, err = opt.kubeClient.CoreV1().Secrets(opt.storageSecret.Namespace).Get(context.TODO(), opt.storageSecret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// if any pre-backup actions has been assigned to it, execute them
	actionOptions := api_util.ActionOptions{
		StashClient:       opt.stashClient,
		TargetRef:         targetRef,
		SetupOptions:      opt.setupOptions,
		BackupSessionName: opt.backupSessionName,
		Namespace:         opt.namespace,
	}
	err = api_util.ExecutePreBackupActions(actionOptions)
	if err != nil {
		return nil, err
	}
	// wait until the backend repository has been initialized.
	err = api_util.WaitForBackendRepository(actionOptions)
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

	port, err := appBinding.Port()
	if err != nil {
		return nil, err
	}

	waitForDBReady(hostname, port, opt.waitTimeout)

	// unmarshal parameter is the field has value
	parameters := v1alpha1.MongoDBConfiguration{}
	if appBinding.Spec.Parameters != nil {
		if err = json.Unmarshal(appBinding.Spec.Parameters.Raw, &parameters); err != nil {
			klog.Errorf("unable to unmarshal appBinding.Spec.Parameters.Raw. Reason: %v", err)
		}
	}

	// Stash operator does not know how many hosts this plugin will backup. It sets totalHosts field of respective BackupSession to 1.
	// We must update the totalHosts field to the actual number of hosts it will backup.
	// Otherwise, BackupSession will stuck in "Running" state.
	// Total hosts for MongoDB:
	// 1. For stand-alone MongoDB, totalHosts=1.
	// 2. For MongoDB ReplicaSet, totalHosts=1.
	// 3. For sharded MongoDB, totalHosts=(number of shard + 1) // extra 1 for config server
	// So, for stand-alone MongoDB and MongoDB ReplicaSet, we don't have to do anything.
	// We only need to update totalHosts field for sharded MongoDB

	// For sharded MongoDB, parameter.ConfigServer will not be empty
	if parameters.ConfigServer != "" {
		backupSession, err := opt.stashClient.StashV1beta1().BackupSessions(opt.namespace).Get(context.TODO(), opt.backupSessionName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		for i, target := range backupSession.Status.Targets {
			if target.Ref.Kind == apis.KindAppBinding && target.Ref.Name == appBinding.Name {
				_, err = stash_cs_util.UpdateBackupSessionStatus(
					context.TODO(),
					opt.stashClient.StashV1beta1(),
					backupSession.ObjectMeta,
					func(status *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
						status.Targets[i].TotalHosts = pointer.Int32P(int32(len(parameters.ReplicaSets) + 1)) // for each shard there will be one key in parameters.ReplicaSet
						return backupSession.UID, status
					},
					metav1.UpdateOptions{},
				)
				if err != nil {
					return nil, err
				}
				break
			}
		}
	}

	if appBinding.Spec.ClientConfig.CABundle != nil {
		if tlsSecret == nil {
			return nil, errors.Wrap(err, "spec.tlsSecret needs to be set in appbinding for TLS secured database.")
		}

		if err := os.WriteFile(filepath.Join(opt.setupOptions.ScratchDir, MongoTLSCertFileName), appBinding.Spec.ClientConfig.CABundle, os.ModePerm); err != nil {
			return nil, err
		}
		mongoCreds = []interface{}{
			"--tls",
			"--tlsCAFile", filepath.Join(opt.setupOptions.ScratchDir, MongoTLSCertFileName),
			"--tlsCertificateKeyFile", filepath.Join(opt.setupOptions.ScratchDir, MongoClientPemFileName),
		}
		dumpCreds = []interface{}{
			"--ssl",
			"--sslCAFile", filepath.Join(opt.setupOptions.ScratchDir, MongoTLSCertFileName),
			"--sslPEMKeyFile", filepath.Join(opt.setupOptions.ScratchDir, MongoClientPemFileName),
		}

		// get certificate secret to get client certificate
		var pemBytes []byte
		var ok bool
		pemBytes, ok = tlsSecret.Data[MongoClientPemFileName]
		if !ok {
			crt, ok := tlsSecret.Data[core.TLSCertKey]
			if !ok {
				fmt.Println("here.........................")
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
		userAuth := []interface{}{
			"-u", user,
			"--authenticationMechanism", "MONGODB-X509",
			"--authenticationDatabase", "$external",
		}
		mongoCreds = append(mongoCreds, userAuth...)
		username := StashUserName
		if parameters.ConfigServer == "" {
			username = string(authSecret.Data[MongoUserKey])
		}
		dumpCreds = append(dumpCreds, []interface{}{
			fmt.Sprintf("--username=%s", username),
			fmt.Sprintf("--password=%s", authSecret.Data[MongoPasswordKey]),
			"--authenticationDatabase", opt.authenticationDatabase,
		}...)

	} else {
		userAuth := []interface{}{
			fmt.Sprintf("--username=%s", authSecret.Data[MongoUserKey]),
			fmt.Sprintf("--password=%s", authSecret.Data[MongoPasswordKey]),
			"--authenticationDatabase", opt.authenticationDatabase,
		}
		mongoCreds = append(mongoCreds, userAuth...)
		username := StashUserName
		if parameters.ConfigServer == "" {
			username = string(authSecret.Data[MongoUserKey])
		}
		dumpCreds = append(dumpCreds, []interface{}{
			fmt.Sprintf("--username=%s", username),
			fmt.Sprintf("--password=%s", authSecret.Data[MongoPasswordKey]),
			"--authenticationDatabase", opt.authenticationDatabase,
		}...)
	}

	getBackupOpt := func(mongoDSN, hostKey string, isStandalone bool) restic.BackupOptions {
		klog.Infoln("processing backupOptions for ", mongoDSN)
		backupOpt := restic.BackupOptions{
			Host:            hostKey,
			StdinFileName:   MongoDumpFile,
			RetentionPolicy: opt.defaultBackupOptions.RetentionPolicy,
			BackupPaths:     opt.defaultBackupOptions.BackupPaths,
		}

		// setup pipe command
		backupCmd := restic.Command{
			Name: MongoDumpCMD,
			Args: append([]interface{}{
				"--host", mongoDSN,
				"--archive",
			}, dumpCreds...),
		}
		userArgs := strings.Fields(opt.mongoArgs)

		if isStandalone {
			backupCmd.Args = append(backupCmd.Args, fmt.Sprintf("--port=%d", port))
		} else {
			// - port is already added in mongoDSN with replicasetName/host:port format.
			// - oplog is enabled automatically for replicasets.
			// Don't use --oplog if user specify any of these arguments through opt.mongoArgs
			// 1. --db
			// 2. --collection
			// xref: https://docs.mongodb.com/manual/reference/program/mongodump/#cmdoption-mongodump-oplog
			forbiddenArgs := sets.NewString(
				"-d", "--db",
				"-c", "--collection",
			)
			if !containsArg(userArgs, forbiddenArgs) {
				backupCmd.Args = append(backupCmd.Args, "--oplog")
			}
		}

		for _, arg := range userArgs {
			backupCmd.Args = append(backupCmd.Args, arg)
		}

		// append the backup command into the pipe
		backupOpt.StdinPipeCommands = append(backupOpt.StdinPipeCommands, backupCmd)
		return backupOpt
	}

	// set opt.maxConcurrency
	if len(parameters.ReplicaSets) <= 1 {
		opt.maxConcurrency = 1
	}

	// If parameters.ReplicaSets is not empty, then replicaset hosts are given in key:value pair,
	// where, keys are host-0,host-1 etc and values are the replicaset dsn of one replicaset component
	//
	// Procedure of taking backup
	// - Disable the balancer from mongos
	// - Lock the secondary component of configserver and replicasets
	// - take backup using dsn and host01, host02 etc (from keys)
	// - Unlock the secondary component for both successful or unsuccessful backup.
	// - enable balancer for both successful or unsuccessful backup
	//
	// ref: https://docs.mongodb.com/manual/tutorial/backup-sharded-cluster-with-database-dumps/

	if parameters.ConfigServer != "" {
		// Workaround for issue: https://jira.mongodb.org/browse/TOOLS-2966
		//
		//	- Check config.reshardingOperations exists from configServer
		//	- if exists:
		//		- check number of documents in config.reshardingOperations collection
		//		- if no documents found:
		//			- rename config.reshardingOperations to config.reshardingOperations_temp
		//			- take backup
		//			- rename back config.reshardingOperations_temp to config.reshardingOperations

		rename, err := handleReshard(parameters.ConfigServer)
		if err != nil {
			return nil, err
		}
		defer func() {
			if rename {
				err := renameTempReshardCollection(parameters.ConfigServer)
				if err != nil {
					klog.Error(err)
				}
			}
		}()

		// We need to create a role and user and backup using that user. This should be removed if the issue is fixed.
		// Issue ref: https://jira.mongodb.org/browse/TOOLS-3203?jql=project%20%3D%20TOOLS%20AND%20component%20%3D%20mongodump
		err = createStashRoleAndUser(parameters.ConfigServer, string(authSecret.Data[MongoPasswordKey]))
		if err != nil {
			klog.Errorf("error while creating user for %v. error: %v", parameters.ConfigServer, err)
			return nil, err
		}

		// sharded cluster. so disable the balancer first. then perform the 'usual' tasks.
		primary, secondary, err := getPrimaryNSecondaryMember(parameters.ConfigServer)
		if err != nil {
			return nil, err
		}

		// connect to mongos to disable/enable balancer
		err = disabelBalancer(hostname)
		cleanupFuncs = append(cleanupFuncs, func() error {
			// even if error occurs, try to enable the balancer on exiting the program.
			return enableBalancer(hostname)
		})
		if err != nil {
			return nil, err
		}

		// backupHost is secondary if any secondary component exists.
		// otherwise primary component will be used to take backup.
		backupHost := primary
		if secondary != "" {
			backupHost = secondary
		}

		err = lockConfigServer(parameters.ConfigServer, secondary)

		cleanupFuncs = append(cleanupFuncs, func() error {
			// even if error occurs, try to unlock the server
			return unlockSecondaryMember(secondary)
		})
		if err != nil {
			klog.Errorf("error while locking config server. error: %v", err)
			return nil, err
		}
		opt.backupOptions = append(opt.backupOptions, getBackupOpt(backupHost, MongoConfigSVRHostKey, false))
	}

	for key, host := range parameters.ReplicaSets {
		if parameters.ConfigServer != "" {
			// We need to create a role and user and backup using that user for shard.
			// These role and user should be removed if the issue is fixed.
			// Issue ref: https://jira.mongodb.org/browse/TOOLS-3203?jql=project%20%3D%20TOOLS%20AND%20component%20%3D%20mongodump
			err = createStashRoleAndUser(host, string(authSecret.Data[MongoPasswordKey]))
			if err != nil {
				klog.Errorf("error while creating user for %v. error: %v", host, err)
				return nil, err
			}
		}

		// do the task
		primary, secondary, err := getPrimaryNSecondaryMember(host)
		if err != nil {
			klog.Errorf("error while getting primary and secondary member of %v. error: %v", host, err)
			return nil, err
		}

		// backupHost is secondary if any secondary component exists.
		// otherwise primary component will be used to take backup.
		backupHost := primary
		if secondary != "" {
			backupHost = secondary
		}

		err = lockSecondaryMember(secondary)
		cleanupFuncs = append(cleanupFuncs, func() error {
			// even if error occurs, try to unlock the server
			return unlockSecondaryMember(secondary)
		})
		if err != nil {
			klog.Errorf("error while locking secondary member %v. error: %v", host, err)
			return nil, err
		}

		opt.backupOptions = append(opt.backupOptions, getBackupOpt(backupHost, key, false))
	}

	// if parameters.ReplicaSets is nil, then the mongodb database doesn't have replicasets or sharded replicasets.
	// In this case, perform normal backup with clientconfig.Service.Name mongo dsn
	if parameters.ReplicaSets == nil {
		opt.backupOptions = append(opt.backupOptions, getBackupOpt(hostname, restic.DefaultHost, true))
	}

	klog.Infoln("processing backup.")

	resticWrapper, err := restic.NewResticWrapper(opt.setupOptions)
	if err != nil {
		return nil, err
	}
	// hide password, don't print cmd
	resticWrapper.HideCMD()

	return resticWrapper.RunParallelBackup(opt.backupOptions, targetRef, opt.maxConcurrency)
}

// cleanup usually unlocks the locked servers
func cleanup() {
	for _, f := range cleanupFuncs {
		if err := f(); err != nil {
			klog.Errorln(err)
		}
	}
}

func getSSLUser(path string) (string, error) {
	data, err := sh.Command(OpenSSLCMD, "x509", "-in", path, "-inform", "PEM", "-subject", "-nameopt", "RFC2253", "-noout").Output()
	if err != nil {
		return "", err
	}

	user := strings.TrimPrefix(string(data), "subject=")
	return strings.TrimSpace(user), nil
}

func getPrimaryNSecondaryMember(mongoDSN string) (primary, secondary string, err error) {
	klog.Infoln("finding primary and secondary instances of", mongoDSN)
	v := make(map[string]interface{})

	// stop balancer
	args := append([]interface{}{
		"config",
		"--host", mongoDSN,
		"--quiet",
		"--eval", "JSON.stringify(rs.isMaster())",
	}, mongoCreds...)
	// even --quiet doesn't skip replicaset PrimaryConnection log. so take tha last line. issue tracker: https://jira.mongodb.org/browse/SERVER-27159
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
		return "", "", err
	}

	primary, ok := v["primary"].(string)
	if !ok || primary == "" {
		return "", "", fmt.Errorf("unable to get primary instance using rs.isMaster(). got response: %v", v)
	}

	hosts, ok := v["hosts"].([]interface{})
	if !ok {
		return "", "", fmt.Errorf("unable to get hosts using rs.isMaster(). got response: %v", v)
	}

	for _, host := range hosts {
		secHost, ok := host.(string)
		if !ok || secHost == "" {
			err = fmt.Errorf("unable to get secondary instance using rs.isMaster(). got response: %v", v)
			continue
		}

		if secHost != primary {
			return primary, secHost, nil
		}
	}

	return primary, "", err
}

// run from mongos instance
func disabelBalancer(mongosHost string) error {
	klog.Infoln("Disabling balancer of ", mongosHost)
	v := make(map[string]interface{})

	args := append([]interface{}{
		"config",
		"--host", mongosHost,
		"--quiet",
		"--eval", "JSON.stringify(sh.stopBalancer())",
	}, mongoCreds...)
	// disable balancer
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
		return err
	}

	if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
		return fmt.Errorf("unable to disable balancer. got response: %v", v)
	}

	// wait for balancer to stop
	args = append([]interface{}{
		"config",
		"--host", mongosHost,
		"--quiet",
		"--eval", "while(sh.isBalancerRunning().mode != 'off'){ print('waiting for balancer to stop...'); sleep(1000);}",
	}, mongoCreds...)
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").Run(); err != nil {
		return err
	}
	return nil
}

func enableBalancer(mongosHost string) error {
	// run separate shell to dump indices
	klog.Infoln("Enabling balancer of ", mongosHost)
	v := make(map[string]interface{})

	// enable balancer
	args := append([]interface{}{
		"config",
		"--host", mongosHost,
		"--quiet",
		"--eval", "JSON.stringify(sh.setBalancerState(true))",
	}, mongoCreds...)
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
		return err
	}

	if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
		return fmt.Errorf("unable to disable balancer. got response: %v", v)
	}

	return nil
}

func lockConfigServer(configSVRDSN, secondaryHost string) error {
	klog.Infoln("Attempting to lock configserver", configSVRDSN)

	if secondaryHost == "" {
		klog.Warningln("locking configserver is skipped. secondary host is empty")
		return nil
	}
	v := make(map[string]interface{})
	// findAndModify BackupControlDocument. skip single quote inside single quote: https://stackoverflow.com/a/28786747/4628962
	args := append([]interface{}{
		"config",
		"--host", configSVRDSN,
		"--quiet",
		"--eval", "JSON.stringify(db.BackupControl.findAndModify({query: { _id: 'BackupControlDocument' }, update: { $inc: { counter : 1 } }, new: true, upsert: true, writeConcern: { w: 'majority', wtimeout: 15000 }}));",
	}, mongoCreds...)
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
		return err
	}
	val, ok := v["counter"].(float64)
	if !ok || int(val) == 0 {
		return fmt.Errorf("unable to modify BackupControlDocument. got response: %v", v)
	}
	val2 := float64(0)
	timer := 0 // wait approximately 5 minutes.
	v2 := make([]map[string]interface{}, 0)
	for timer < 60 && (int(val2) == 0 || int(val) != int(val2)) {
		timer++
		// find backupDocument from secondary configServer
		args = append([]interface{}{
			"config",
			"--host", secondaryHost,
			"--quiet",
			"--eval", "rs.secondaryOk(); JSON.stringify(db.BackupControl.find({ '_id' : 'BackupControlDocument' }).readConcern('majority').toArray());",
		}, mongoCreds...)

		if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v2); err != nil {
			return err
		}
		if len(v2) > 0 {
			val2, ok = v2[0]["counter"].(float64)
			if !ok {
				return fmt.Errorf("unable to get BackupControlDocument. got response: %v", v)
			}
		}
		if int(val) != int(val2) {
			klog.V(5).Infof("BackupDocument counter in secondary is not same. Expected %v, but got %v. Full response: %v", val, val2, v)
			time.Sleep(time.Second * 5)
		}
	}
	if timer >= 60 {
		return fmt.Errorf("timeout while waiting for BackupDocument counter in secondary to be same as primary. Expected %v, but got %v. Full response: %v", val, val2, v)
	}
	// lock secondary
	return lockSecondaryMember(secondaryHost)
}

func lockSecondaryMember(mongohost string) error {
	klog.Infoln("Attempting to lock secondary member", mongohost)
	if mongohost == "" {
		klog.Warningln("locking secondary member is skipped. secondary host is empty")
		return nil
	}
	v := make(map[string]interface{})

	// lock file
	args := append([]interface{}{
		"config",
		"--host", mongohost,
		"--quiet",
		"--eval", "JSON.stringify(db.fsyncLock())",
	}, mongoCreds...)
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
		return err
	}

	if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
		return fmt.Errorf("unable to lock the secondary host. got response: %v", v)
	}

	return nil
}

func unlockSecondaryMember(mongohost string) error {
	klog.Infoln("Attempting to unlock secondary member", mongohost)
	if mongohost == "" {
		klog.Warningln("skipped unlocking secondary member. secondary host is empty")
		return nil
	}
	v := make(map[string]interface{})

	// unlock file
	args := append([]interface{}{
		"config",
		"--host", mongohost,
		"--quiet",
		"--eval", "JSON.stringify(db.fsyncUnlock())",
	}, mongoCreds...)
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
		return err
	}

	if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
		return fmt.Errorf("unable to lock the secondary host. got response: %v", v)
	}

	return nil
}

func checkRoleExists(mongoDSN string) (bool, error) {
	v := make(map[string]interface{})
	args := append([]interface{}{
		"admin",
		"--host", mongoDSN,
		"--quiet",
		"--eval", `JSON.stringify(db.getRole("` + StashRoleName + `"))`,
	}, mongoCreds...)
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
		return false, err
	}

	if val, ok := v["role"].(string); ok && string(val) == StashRoleName {
		return true, nil
	}

	return false, nil
}

func checkUserExists(mongoDSN string) (bool, error) {
	v := make(map[string]interface{})
	args := append([]interface{}{
		"admin",
		"--host", mongoDSN,
		"--quiet",
		"--eval", `JSON.stringify(db.getUser("` + StashUserName + `"))`,
	}, mongoCreds...)
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
		return false, err
	}

	if val, ok := v["user"].(string); ok && string(val) == StashUserName {
		return true, nil
	}

	return false, nil
}

func createStashRoleAndUser(mongoDSN string, pass string) error {
	err := createStashBackupRole(mongoDSN)
	if err != nil {
		return err
	}

	return createStashBackupUser(mongoDSN, pass)
}

func createStashBackupRole(mongoDSN string) error {
	exists, err := checkRoleExists(mongoDSN)
	if err != nil {
		return err
	}
	if !exists {
		klog.Infoln("creating role " + StashRoleName)
		v := make(map[string]interface{})

		args := append([]interface{}{
			"admin",
			"--host", mongoDSN,
			"--quiet",
			"--eval", `JSON.stringify(db.runCommand({createRole: "` + StashRoleName + `",privileges:[{resource:{db:"config",collection:"system.preimages"},actions:["find"]},{resource:{db:"config",collection:"system.sharding_ddl_coordinators"},actions:["find"]},{resource:{db:"config",collection:"system.*"},actions:["find"]}],roles: []}))`,
		}, mongoCreds...)

		if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
			return err
		}

		if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
			return fmt.Errorf("unable to create role %v. got response: %v", StashRoleName, v)
		}
	}

	return nil
}

func createStashBackupUser(mongoDSN string, pass string) error {
	exists, err := checkUserExists(mongoDSN)
	if err != nil {
		return err
	}
	if !exists {
		klog.Infoln("creating user " + StashUserName)
		v := make(map[string]interface{})

		args := append([]interface{}{
			"admin",
			"--host", mongoDSN,
			"--quiet",
			"--eval", `JSON.stringify(db.runCommand({createUser: "` + StashUserName + `" ,pwd: "` + pass + `", roles:[{role:"backup", db:"admin"}, {role: "` + StashRoleName + `",db:"admin"}]}))`,
		}, mongoCreds...)
		if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
			return err
		}

		if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
			return fmt.Errorf("unable to create user %v. got response: %v", StashUserName, v)
		}
	}
	return nil
}

func handleReshard(configsvrDSN string) (bool, error) {
	v := make([]interface{}, 0)
	args := append([]interface{}{
		"config",
		"--host", configsvrDSN,
		"--quiet",
		"--eval", `JSON.stringify(db.getCollectionNames())`,
	}, mongoCreds...)
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
		return false, err
	}

	exists := false
	for _, name := range v {
		if name.(string) == "reshardingOperations" {
			exists = true
			break
		}
	}
	if !exists {
		return false, nil
	}

	args = append([]interface{}{
		"config",
		"--host", configsvrDSN,
		"--quiet",
		"--eval", `JSON.stringify(db.reshardingOperations.count())`,
	}, mongoCreds...)
	out, err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").Output()
	if err != nil {
		return false, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return false, err
	}
	if count != 0 {
		return false, nil
	}

	res := make(map[string]interface{})
	args = append([]interface{}{
		"config",
		"--host", configsvrDSN,
		"--quiet",
		"--eval", `JSON.stringify(db.adminCommand( { renameCollection: "config.reshardingOperations", to: "config.reshardingOperations_temp", dropTarget: true}))`,
	}, mongoCreds...)
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&res); err != nil {
		return false, err
	}
	if val, ok := res["ok"].(float64); !ok || int(val) != 1 {
		return false, fmt.Errorf("unable to rename collection config.reshardingOperations. got response: %v", res)
	}

	return true, nil
}

func renameTempReshardCollection(configsvrDSN string) error {
	res := make(map[string]interface{})
	args := append([]interface{}{
		"config",
		"--host", configsvrDSN,
		"--quiet",
		"--eval", `JSON.stringify(db.adminCommand( { renameCollection: "config.reshardingOperations_temp", to: "config.reshardingOperations" } ))`,
	}, mongoCreds...)
	if err := sh.Command(MongoshCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&res); err != nil {
		return err
	}
	if val, ok := res["ok"].(float64); !ok || int(val) != 1 {
		return fmt.Errorf("unable to rename collection config.reshardingOperations_temp. got response: %v", res)
	}

	return nil
}
