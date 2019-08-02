package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	"github.com/codeskyblue/go-sh"
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

const (
	JobMongoBackup        = "stash-mongo-backup"
	MongoUserKey          = "username"
	MongoPasswordKey      = "password"
	MongoDumpFile         = "dump"
	MongoDumpCMD          = "mongodump"
	MongoRestoreCMD       = "mongorestore"
	MongoConfigSVRHostKey = "confighost"

	MongoTLSCertFileName   = "ca.cert"
	MongoClientPemFileName = "client.pem"
)

var (
	MongoCMD     = "/usr/bin/mongo"
	OpenSSLCMD   = "/usr/bin/openssl"
	adminCreds   []interface{}
	cleanupFuncs []func() error
)

func init() {
	var err error
	if MongoCMD, err = exec.LookPath("mongo"); err != nil {
		log.Fatalln("unable to look for mongo command. reason:", err)
	}
	if OpenSSLCMD, err = exec.LookPath("openssl"); err != nil {
		log.Fatalln("unable to look for openssl command. reason:", err)
	}
}

func NewCmdBackup() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		namespace      string
		appBindingName string
		mongoArgs      string
		outputDir      string
		maxConcurrency int

		backupOpts []restic.BackupOptions
		setupOpt   = restic.SetupOptions{
			ScratchDir:  restic.DefaultScratchDir,
			EnableCache: false,
		}
		defaultBackupOpt = restic.BackupOptions{
			Host:          restic.DefaultHost,
			StdinFileName: MongoDumpFile,
		}
		metrics = restic.MetricsOptions{
			JobName: JobMongoBackup,
		}
	)

	cmd := &cobra.Command{
		Use:               "backup-mongo",
		Short:             "Takes a backup of Mongo DB",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			defer cleanup()

			flags.EnsureRequiredFlags(cmd, "app-binding", "provider", "secret-dir")

			// catch sigkill signals and gracefully terminate so that cleanup functions are executed.
			sigChan := make(chan os.Signal)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				rcvSig := <-sigChan
				cleanup()
				log.Errorf("Received signal: %s, exiting\n", rcvSig)
				os.Exit(1)
			}()

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

			// wait for DB ready
			waitForDBReady(appBinding.Spec.ClientConfig.Service.Name, appBinding.Spec.ClientConfig.Service.Port)

			// unmarshal parameter is the field has value
			parameters := v1alpha1.MongoDBConfiguration{}
			if appBinding.Spec.Parameters != nil {
				if err = json.Unmarshal(appBinding.Spec.Parameters.Raw, &parameters); err != nil {
					log.Errorf("unable to unmarshal appBinding.Spec.Parameters.Raw. Reason: %v", err)
				}
			}

			if appBinding.Spec.ClientConfig.CABundle != nil {
				if err := ioutil.WriteFile(filepath.Join(setupOpt.ScratchDir, MongoTLSCertFileName), appBinding.Spec.ClientConfig.CABundle, os.ModePerm); err != nil {
					return errors.Wrap(err, "failed to write key for CA certificate")
				}
				adminCreds = []interface{}{
					"--ssl",
					"--sslCAFile", filepath.Join(setupOpt.ScratchDir, MongoTLSCertFileName),
				}

				// get certificate secret to get client certificate
				data, ok := appBindingSecret.Data[MongoClientPemFileName]
				if !ok {
					return errors.Wrap(err, "unable to get client certificate from secret.")
				}
				if err := ioutil.WriteFile(filepath.Join(setupOpt.ScratchDir, MongoClientPemFileName), data, os.ModePerm); err != nil {
					return errors.Wrap(err, "failed to write client certificate")
				}
				user, err := getSSLUser(filepath.Join(setupOpt.ScratchDir, MongoClientPemFileName))
				if err != nil {
					return errors.Wrap(err, "unable to get user from ssl.")
				}
				adminCreds = append(adminCreds, []interface{}{
					"--sslPEMKeyFile", filepath.Join(setupOpt.ScratchDir, MongoClientPemFileName),
					"-u", user,
					"--authenticationMechanism", "MONGODB-X509",
					"--authenticationDatabase", "$external",
				}...)

			} else {
				adminCreds = []interface{}{
					"--username", string(appBindingSecret.Data[MongoUserKey]),
					"--password", string(appBindingSecret.Data[MongoPasswordKey]),
					"--authenticationDatabase", "admin",
				}
			}

			getBackupOpt := func(mongoDSN, hostKey string, isStandalone bool) restic.BackupOptions {
				log.Infoln("processing backupOptions for ", mongoDSN)
				backupOpt := restic.BackupOptions{
					Host:            hostKey,
					StdinFileName:   MongoDumpFile,
					RetentionPolicy: defaultBackupOpt.RetentionPolicy,
					BackupPaths:     defaultBackupOpt.BackupPaths,
				}

				// setup pipe command
				backupOpt.StdinPipeCommand = restic.Command{
					Name: MongoDumpCMD,
					Args: append([]interface{}{
						"--host", mongoDSN,
						"--archive",
					}, adminCreds...),
				}
				if isStandalone {
					backupOpt.StdinPipeCommand.Args = append(backupOpt.StdinPipeCommand.Args, "--port="+fmt.Sprint(appBinding.Spec.ClientConfig.Service.Port))
				} else {
					// - port is already added in mongoDSN with replicasetName/host:port format.
					// - oplog is enabled automatically for replicasets.
					backupOpt.StdinPipeCommand.Args = append(backupOpt.StdinPipeCommand.Args, "--oplog")
				}
				if mongoArgs != "" {
					backupOpt.StdinPipeCommand.Args = append(backupOpt.StdinPipeCommand.Args, mongoArgs)
				}
				return backupOpt
			}

			// set maxConcurrency
			if len(parameters.ReplicaSets) <= 1 {
				maxConcurrency = 1
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
				// sharded cluster. so disable the balancer first. then perform the 'usual' tasks.

				primary, secondary, err := getPrimaryNSecondaryMember(parameters.ConfigServer)
				if err != nil {
					return err
				}

				// connect to mongos to disable/enable balancer
				err = disabelBalancer(appBinding.Spec.ClientConfig.Service.Name)
				cleanupFuncs = append(cleanupFuncs, func() error {
					// even if error occurs, try to enable the balancer on exiting the program.
					return enableBalancer(appBinding.Spec.ClientConfig.Service.Name)
				})
				if err != nil {
					return err
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
					return err
				}

				backupOpts = append(backupOpts, getBackupOpt(backupHost, MongoConfigSVRHostKey, false))
			}

			for key, host := range parameters.ReplicaSets {
				// do the task
				primary, secondary, err := getPrimaryNSecondaryMember(host)
				if err != nil {
					log.Errorf("error while getting primary and secondary member of %v. error: %v", host, err)
					return err
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
					log.Errorf("error while locking secondary member %v. error: %v", host, err)
					return err
				}

				backupOpts = append(backupOpts, getBackupOpt(backupHost, key, false))
			}

			// if parameters.ReplicaSets is nil, then the mongodb database doesn't have replicasets or sharded replicasets.
			// In this case, perform normal backup with clientconfig.Service.Name mongo dsn
			if parameters.ReplicaSets == nil {
				backupOpts = append(backupOpts, getBackupOpt(appBinding.Spec.ClientConfig.Service.Name, restic.DefaultHost, true))
			}

			log.Infoln("processing backup.")

			resticWrapper, err := restic.NewResticWrapper(setupOpt)
			if err != nil {
				return err
			}
			// hide password, don't print cmd
			resticWrapper.HideCMD()

			// Run backup
			backupOutput, backupErr := resticWrapper.RunParallelBackup(backupOpts, maxConcurrency)
			// If metrics are enabled then generate metrics
			if metrics.Enabled {
				err := backupOutput.HandleMetrics(&metrics, backupErr)
				if err != nil {
					return kerrors.NewAggregate([]error{backupErr, err})
				}
			}
			// If output directory specified, then write the output in "output.json" file in the specified directory
			if backupErr == nil && outputDir != "" {
				err := backupOutput.WriteOutput(filepath.Join(outputDir, restic.DefaultOutputFileName))
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

	cmd.Flags().StringVar(&defaultBackupOpt.Host, "hostname", defaultBackupOpt.Host, "Name of the host machine")

	cmd.Flags().IntVar(&defaultBackupOpt.RetentionPolicy.KeepLast, "retention-keep-last", defaultBackupOpt.RetentionPolicy.KeepLast, "Specify value for retention strategy")
	cmd.Flags().IntVar(&defaultBackupOpt.RetentionPolicy.KeepHourly, "retention-keep-hourly", defaultBackupOpt.RetentionPolicy.KeepHourly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&defaultBackupOpt.RetentionPolicy.KeepDaily, "retention-keep-daily", defaultBackupOpt.RetentionPolicy.KeepDaily, "Specify value for retention strategy")
	cmd.Flags().IntVar(&defaultBackupOpt.RetentionPolicy.KeepWeekly, "retention-keep-weekly", defaultBackupOpt.RetentionPolicy.KeepWeekly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&defaultBackupOpt.RetentionPolicy.KeepMonthly, "retention-keep-monthly", defaultBackupOpt.RetentionPolicy.KeepMonthly, "Specify value for retention strategy")
	cmd.Flags().IntVar(&defaultBackupOpt.RetentionPolicy.KeepYearly, "retention-keep-yearly", defaultBackupOpt.RetentionPolicy.KeepYearly, "Specify value for retention strategy")
	cmd.Flags().StringSliceVar(&defaultBackupOpt.RetentionPolicy.KeepTags, "retention-keep-tags", defaultBackupOpt.RetentionPolicy.KeepTags, "Specify value for retention strategy")
	cmd.Flags().BoolVar(&defaultBackupOpt.RetentionPolicy.Prune, "retention-prune", defaultBackupOpt.RetentionPolicy.Prune, "Specify whether to prune old snapshot data")
	cmd.Flags().BoolVar(&defaultBackupOpt.RetentionPolicy.DryRun, "retention-dry-run", defaultBackupOpt.RetentionPolicy.DryRun, "Specify whether to test retention policy without deleting actual data")

	cmd.Flags().StringVar(&outputDir, "output-dir", outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	cmd.Flags().BoolVar(&metrics.Enabled, "metrics-enabled", metrics.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&metrics.PushgatewayURL, "metrics-pushgateway-url", metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringVar(&metrics.MetricFileDir, "metrics-dir", metrics.MetricFileDir, "Directory where to write metric.prom file (keep empty if you don't want to write metric in a text file)")
	cmd.Flags().StringSliceVar(&metrics.Labels, "metrics-labels", metrics.Labels, "Labels to apply in exported metrics")

	return cmd
}

// cleanup usually unlocks the locked servers
func cleanup() {
	for _, f := range cleanupFuncs {
		if err := f(); err != nil {
			log.Errorln(err)
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
	log.Infoln("finding primary and secondary instances of", mongoDSN)
	v := make(map[string]interface{})

	//stop balancer
	args := append([]interface{}{
		"config",
		"--host", mongoDSN,
		"--quiet",
		"--eval", "JSON.stringify(rs.isMaster())",
	}, adminCreds...)
	// even --quiet doesn't skip replicaset PrimaryConnection log. so take tha last line. issue tracker: https://jira.mongodb.org/browse/SERVER-27159
	if err := sh.Command(MongoCMD, args...).Command("/usr/bin/tail", "-1").UnmarshalJSON(&v); err != nil {
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
	log.Infoln("Disabling balancer of ", mongosHost)
	v := make(map[string]interface{})

	args := append([]interface{}{
		"config",
		"--host", mongosHost,
		"--quiet",
		"--eval", "JSON.stringify(sh.stopBalancer())",
	}, adminCreds...)
	// disable balancer
	if err := sh.Command(MongoCMD, args...).UnmarshalJSON(&v); err != nil {
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
		"--eval", "while(sh.isBalancerRunning()){ print('waiting for balancer to stop...'); sleep(1000);}",
	}, adminCreds...)
	if err := sh.Command(MongoCMD, args...).Run(); err != nil {
		return err
	}
	return nil
}

func enableBalancer(mongosHost string) error {
	// run separate shell to dump indices
	log.Infoln("Enabling balancer of ", mongosHost)
	v := make(map[string]interface{})

	// enable balancer
	args := append([]interface{}{
		"config",
		"--host", mongosHost,
		"--quiet",
		"--eval", "JSON.stringify(sh.setBalancerState(true))",
	}, adminCreds...)
	if err := sh.Command(MongoCMD, args...).UnmarshalJSON(&v); err != nil {
		return err
	}

	if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
		return fmt.Errorf("unable to disable balancer. got response: %v", v)
	}

	return nil
}

func lockConfigServer(configSVRDSN, secondaryHost string) error {
	log.Infoln("Attempting to lock configserver", configSVRDSN)
	if secondaryHost == "" {
		log.Warningln("locking configserver is skipped. secondary host is empty")
		return nil
	}
	v := make(map[string]interface{})

	// findAndModify BackupControlDocument. skip single quote inside single quote: https://stackoverflow.com/a/28786747/4628962
	args := append([]interface{}{
		"config",
		"--host", configSVRDSN,
		"--quiet",
		"--eval", "db.BackupControl.findAndModify({query: { _id: 'BackupControlDocument' }, update: { $inc: { counter : 1 } }, new: true, upsert: true, writeConcern: { w: 'majority', wtimeout: 15000 }});",
	}, adminCreds...)
	if err := sh.Command(MongoCMD, args...).Command("tail", "-1").UnmarshalJSON(&v); err != nil {
		return err
	}

	val, ok := v["counter"].(float64)
	if !ok || int(val) == 0 {
		return fmt.Errorf("unable to modify BackupControlDocument. got response: %v", v)
	}

	val2 := float64(0)
	timer := 0 // wait approximately 5 minutes.
	for timer < 60 && (int(val2) == 0 || int(val) != int(val2)) {
		timer++
		// find backupDocument from secondary configServer
		args = append([]interface{}{
			"config",
			"--host", secondaryHost,
			"--quiet",
			"--eval", "rs.slaveOk(); db.BackupControl.find({ '_id' : 'BackupControlDocument' }).readConcern('majority');",
		}, adminCreds...)
		if err := sh.Command(MongoCMD, args...).UnmarshalJSON(&v); err != nil {
			return err
		}

		val2, ok = v["counter"].(float64)
		if !ok {
			return fmt.Errorf("unable to get BackupControlDocument. got response: %v", v)
		}

		if int(val) != int(val2) {
			log.Debugf("BackupDocument counter in secondary is not same. Expected %v, but got %v. Full response: %v", val, val2, v)
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
	log.Infoln("Attempting to lock secondary member", mongohost)
	if mongohost == "" {
		log.Warningln("locking secondary member is skipped. secondary host is empty")
		return nil
	}
	v := make(map[string]interface{})

	// lock file
	args := append([]interface{}{
		"config",
		"--host", mongohost,
		"--quiet",
		"--eval", "JSON.stringify(db.fsyncLock())",
	}, adminCreds...)
	if err := sh.Command(MongoCMD, args...).UnmarshalJSON(&v); err != nil {
		return err
	}

	if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
		return fmt.Errorf("unable to lock the secondary host. got response: %v", v)
	}

	return nil
}

func unlockSecondaryMember(mongohost string) error {
	log.Infoln("Attempting to unlock secondary member", mongohost)
	if mongohost == "" {
		log.Warningln("skipped unlocking secondary member. secondary host is empty")
		return nil
	}
	v := make(map[string]interface{})

	// unlock file
	args := append([]interface{}{
		"config",
		"--host", mongohost,
		"--quiet",
		"--eval", "JSON.stringify(db.fsyncUnlock())",
	}, adminCreds...)
	if err := sh.Command(MongoCMD, args...).UnmarshalJSON(&v); err != nil {
		return err
	}

	if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
		return fmt.Errorf("unable to lock the secondary host. got response: %v", v)
	}

	return nil
}
