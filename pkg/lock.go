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
	"encoding/json"
	"fmt"
	"time"

	"gomodules.xyz/go-sh"
	"k8s.io/klog/v2"
)

func setupConfigServer(configSVRDSN, secondaryHost string) error {
	klog.Infof("Attempting to setup configserver %s\n", configSVRDSN)

	if secondaryHost == "" {
		klog.Warningln("locking configserver is skipped. secondary host is empty")
		return nil
	}
	v := make(map[string]any)
	// findAndModify BackupControlDocument. skip single quote inside single quote: https://stackoverflow.com/a/28786747/4628962
	args := append([]any{
		"config",
		"--host", configSVRDSN,
		"--quiet",
		"--eval", "db.BackupControl.findAndModify({query: { _id: 'BackupControlDocument' }, update: { $inc: { counter : 1 } }, new: true, upsert: true, writeConcern: { w: 'majority', wtimeout: 15000 }});",
	}, mongoCreds...)

	output, err := sh.Command(MongoCMD, args...).Command("/usr/bin/tail", "-1").Output()
	if err != nil {
		klog.Errorf("Error while running findAndModify to setup configServer : %s ; output : %s \n", err.Error(), output)
		return err
	}

	output, err = extractJSON(string(output))
	if err != nil {
		return err
	}

	err = json.Unmarshal(output, &v)
	if err != nil {
		klog.Errorf("Unmarshal error while running findAndModify to setup configServer : %s \n", err.Error())
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
		args = append([]any{
			"config",
			"--host", secondaryHost,
			"--quiet",
			"--eval", "rs.secondaryOk(); db.BackupControl.find({ '_id' : 'BackupControlDocument' }).readConcern('majority');",
		}, mongoCreds...)

		output, err := sh.Command(MongoCMD, args...).Command("/usr/bin/tail", "-1").Output()
		if err != nil {
			return err
		}

		output, err = extractJSON(string(output))
		if err != nil {
			return err
		}

		err = json.Unmarshal(output, &v)
		if err != nil {
			return err
		}

		val2, ok = v["counter"].(float64)
		if !ok {
			return fmt.Errorf("unable to get BackupControlDocument. got response: %v", v)
		}
		if int(val) != int(val2) {
			klog.V(5).Infof("BackupDocument counter in secondary %v is not same. Expected %v, but got %v. Full response: %v", secondaryHost, val, val2, v)
			time.Sleep(time.Second * 5)
		}
	}
	if timer >= 60 {
		return fmt.Errorf("timeout while waiting for BackupDocument counter in secondary %v to be same as primary. Expected %v, but got %v. Full response: %v", secondaryHost, val, val2, v)
	}

	return nil
}

func lockSecondaryMember(mongohost string) error {
	klog.Infof("Attempting to lock secondary member %s\n", mongohost)

	if mongohost == "" {
		klog.Warningln("locking secondary member is skipped. secondary host is empty")
		return nil
	}

	// lock file
	v := make(map[string]any)
	args := append([]any{
		"config",
		"--host", mongohost,
		"--quiet",
		"--eval", "JSON.stringify(db.fsyncLock())",
	}, mongoCreds...)

	output, err := sh.Command(MongoCMD, args...).Output()
	if err != nil {
		klog.Errorf("Error while running fsyncLock on secondary : %s ; output : %s \n", err.Error(), output)
		return err
	}

	output, err = extractJSON(string(output))
	if err != nil {
		return err
	}

	err = json.Unmarshal(output, &v)
	if err != nil {
		klog.Errorf("Unmarshal error while running fsyncLock on secondary : %s \n", err.Error())
		return err
	}

	if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
		return fmt.Errorf("unable to lock the secondary host. got response: %v", v)
	}
	klog.Infof("secondary %s locked.\n", mongohost)
	return nil
}

func checkIfSecondaryLockedAndSync(mongohost string) error {
	klog.Infof("Checking if secondary %s is already locked\n", mongohost)

	x := make(map[string]any)
	args := append([]any{
		"config",
		"--host", mongohost,
		"--quiet",
		"--eval", "rs.secondaryOk(); JSON.stringify(db.runCommand({currentOp:1}))",
	}, mongoCreds...)
	output, err := sh.Command(MongoCMD, args...).Output()
	if err != nil {
		klog.Errorf("Error while running currentOp on secondary : %s ; output : %s \n", err.Error(), output)
		return err
	}

	output, err = extractJSON(string(output))
	if err != nil {
		return err
	}

	err = json.Unmarshal(output, &x)
	if err != nil {
		klog.Errorf("Unmarshal error while running currentOp on secondary : %s \n", err.Error())
		return err
	}

	val, ok := x["fsyncLock"].(bool)
	if ok && val {
		klog.Infoln("Found fsyncLock true while locking")
		err := unlockSecondaryMember(mongohost)
		if err != nil {
			return err
		}
		if err := waitForSecondarySync(mongohost); err != nil {
			return err
		}
	}
	return nil
}

func waitForSecondarySync(mongohost string) error {
	klog.Infof("Attempting to sync secondary %s with primary\n", mongohost)

	for {
		status := make(map[string]any)
		args := append([]any{
			"config",
			"--host", mongohost,
			"--quiet",
			"--eval", "JSON.stringify(rs.status())",
		}, mongoCreds...)

		output, err := sh.Command(MongoCMD, args...).Command("/usr/bin/tail", "-1").Output()
		if err != nil {
			return err
		}

		output, err = extractJSON(string(output))
		if err != nil {
			return err
		}

		err = json.Unmarshal(output, &status)
		if err != nil {
			return err
		}

		members, ok := status["members"].([]any)
		if !ok {
			return fmt.Errorf("unable to get members using rs.status(). got response: %v", status)
		}

		var masterOptimeDate, curOptimeDate time.Time

		for _, member := range members {

			memberInfo, ok := member.(map[string]any)
			if !ok {
				return fmt.Errorf("unable to get member info of primary using rs.status(). got response: %v", member)
			}

			if memberInfo["stateStr"] == "PRIMARY" {
				optimedate, ok := memberInfo["optimeDate"].(string)
				if !ok {
					return fmt.Errorf("unable to get optimedate of primary using rs.status(). got response: %v", memberInfo)
				}

				convTime, err := getTime(optimedate)
				if err != nil {
					return err
				}
				masterOptimeDate = convTime
				break
			}
		}
		synced := true
		for _, member := range members {

			memberInfo, ok := member.(map[string]any)
			if !ok {
				return fmt.Errorf("unable to get member info of secondary using rs.status(). got response: %v", member)
			}

			if memberInfo["stateStr"] == "SECONDARY" && memberInfo["name"] == mongohost {

				optimedate, ok := memberInfo["optimeDate"].(string)
				if !ok {
					return fmt.Errorf("unable to get optimedate of secondary using rs.status(). got response: %v", memberInfo)
				}

				convTime, err := getTime(optimedate)
				if err != nil {
					return err
				}
				curOptimeDate = convTime
				if curOptimeDate.Before(masterOptimeDate) {
					synced = false
				}
				break
			}
		}
		if synced {
			klog.Infoln("database successfully synced")
			break
		}

		klog.Infoln("Waiting... database is not synced yet")
		time.Sleep(5 * time.Second)
	}
	return nil
}

func unlockSecondaryMember(mongohost string) error {
	klog.Infof("Attempting to unlock secondary member %s\n", mongohost)
	if mongohost == "" {
		klog.Warningln("skipped unlocking secondary member. secondary host is empty")
		return nil
	}
	v := make(map[string]any)

	// unlock file
	args := append([]any{
		"config",
		"--host", mongohost,
		"--quiet",
		"--eval", "JSON.stringify(db.fsyncUnlock())",
	}, mongoCreds...)
	output, err := sh.Command(MongoCMD, args...).Output()
	if err != nil {
		klog.Errorf("Error while running fsyncUnlock on secondary : %s ; output : %s \n", err.Error(), output)
		return err
	}

	output, err = extractJSON(string(output))
	if err != nil {
		return err
	}

	err = json.Unmarshal(output, &v)
	if err != nil {
		klog.Errorf("Unmarshal error while running fsyncUnlock on secondary : %s \n", err.Error())
		return err
	}
	if val, ok := v["ok"].(float64); !ok || int(val) != 1 {
		return fmt.Errorf("unable to lock the secondary host. got response: %v", v)
	}
	klog.Infof("secondary %s unlocked\n", mongohost)
	return nil
}
