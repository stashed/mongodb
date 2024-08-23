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
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"time"

	stash_cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/restic"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

const (
	MongoUserKey          = "username"
	MongoPasswordKey      = "password"
	MongoDumpFile         = "dump"
	MongoDumpCMD          = "mongodump"
	MongoRestoreCMD       = "mongorestore"
	MongoConfigSVRHostKey = "confighost"

	MongoTLSCertFileName   = "ca.cert"
	MongoClientPemFileName = "client.pem"
)

type mongoOptions struct {
	kubeClient    kubernetes.Interface
	catalogClient appcatalog_cs.Interface
	stashClient   stash_cs.Interface

	namespace              string
	backupSessionName      string
	restoreSessionName     string
	appBindingName         string
	appBindingNamespace    string
	mongoArgs              string
	maxConcurrency         int
	waitTimeout            int32
	outputDir              string
	storageSecret          kmapi.ObjectReference
	authenticationDatabase string

	setupOptions         restic.SetupOptions
	backupOptions        []restic.BackupOptions
	defaultBackupOptions restic.BackupOptions
	dumpOptions          []restic.DumpOptions
	defaultDumpOptions   restic.DumpOptions
	config               *restclient.Config
	totalHosts           int
}

func waitForDBReady(host string, port, waitTimeout int32) {
	klog.Infoln("Checking database connection")
	cmd := fmt.Sprintf(`nc "%s" "%d" -w %d`, host, port, waitTimeout)
	for {
		if err := exec.Command(cmd).Run(); err != nil {
			break
		}
		klog.Infoln("Waiting... database is not ready yet")
		time.Sleep(5 * time.Second)
	}
}

func containsArg(args []string, checklist sets.Set[string]) bool {
	for i := range args {
		a := strings.FieldsFunc(args[i], func(r rune) bool {
			return r == '='
		})
		if checklist.Has(a[0]) {
			return true
		}
	}
	return false
}

func containsString(a []string, e string) bool {
	for _, s := range a {
		if s == e {
			return true
		}
	}
	return false
}

func getTime(t string) (time.Time, error) {
	// Define the layout or format of the input string
	layout := "2006-01-02T15:04:05Z"

	parsedTime, err := time.Parse(layout, t)
	if err != nil {
		return time.Time{}, err
	}
	return parsedTime, nil
}

func isSrvConnection(connectionString string) (bool, error) {
	parsedURL, err := url.Parse(connectionString)
	if err != nil {
		return false, err
	}

	// Check if the scheme is "mongodb+srv"
	return parsedURL.Scheme == "mongodb+srv", nil
}

func (opt *mongoOptions) buildMongoURI(mongoDSN string, port int32, isStandalone, isSrv, tlsEnable bool) string {
	prefix := "mongodb"
	portStr := fmt.Sprintf(":%d", port)
	if isSrv {
		prefix += "+srv"
		portStr = ""
	}
	if !isStandalone {
		portStr = ""
	}

	authDbName := getOptionValue(dumpCreds, "--authenticationDatabase")
	userName := getOptionValue(dumpCreds, "--username")
	if !tlsEnable {
		password := getOptionValue(dumpCreds, "--password")
		return fmt.Sprintf("%s://%s:%s@%s%s/dbnew?authSource=%s",
			prefix, userName, password, mongoDSN, portStr, authDbName)
	}

	authMechanism := getOptionValue(dumpCreds, "--authenticationMechanism")
	return fmt.Sprintf("%s://%s@%s%s/dbnew?authSource=%s&authMechanism=%s&ssl=true",
		prefix, userName, mongoDSN, portStr, authDbName, authMechanism)
}
