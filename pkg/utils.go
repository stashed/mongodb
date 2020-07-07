/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pkg

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	stash_cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/restic"

	"github.com/appscode/go/log"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
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

	namespace          string
	backupSessionName  string
	restoreSessionName string
	appBindingName     string
	mongoArgs          string
	maxConcurrency     int
	waitTimeout        int32
	outputDir          string

	setupOptions         restic.SetupOptions
	backupOptions        []restic.BackupOptions
	defaultBackupOptions restic.BackupOptions
	dumpOptions          []restic.DumpOptions
	defaultDumpOptions   restic.DumpOptions
}

func waitForDBReady(host string, port, waitTimeout int32) {
	log.Infoln("Checking database connection")
	cmd := fmt.Sprintf(`nc "%s" "%d" -w %d`, host, port, waitTimeout)
	for {
		if err := exec.Command(cmd).Run(); err != nil {
			break
		}
		log.Infoln("Waiting... database is not ready yet")
		time.Sleep(5 * time.Second)
	}
}

func containsArg(args []string, checklist sets.String) bool {
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
