---
title: Backup MongoDB | Stash
description: Backup MongoDB database using Stash
menu:
  product_stash_0.8.3:
    identifier: database-mongodb
    name: MongoDB
    parent: database
    weight: 20
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
---

# Backup and Restore MongoDB database using Stash

Stash 0.9.0+ supports backup and restoration of MongoDB databases. This guide will show you how you can backup and restore your MongoDB database with Stash.

## Before You Begin

- At first, you need to have a Kubernetes cluster, and the `kubectl` command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using Minikube.

- Install Stash in your cluster following the steps [here](https://appscode.com/products/stash/0.8.3/setup/install/).

- Install [KubeDB](https://kubedb.com) in your cluster following the steps [here](https://kubedb.com/docs/0.12.0/setup/install/). This step is optional. You can deploy your database using any method you want. We are using KubeDB because it automates some tasks that you have to do manually otherwise.

- If you are not familiar with how Stash backup and restore databases, please check the following guide:
  - [How Stash backup and restore databases](https://appscode.com/products/stash/0.8.3/guides/databases/overview/).

You have to be familiar with following custom resources:

- [AppBinding](https://appscode.com/products/stash/0.8.3/concepts/crds/appbinding/)
- [Function](https://appscode.com/products/stash/0.8.3/concepts/crds/function/)
- [Task](https://appscode.com/products/stash/0.8.3/concepts/crds/task/)
- [BackupConfiguration](https://appscode.com/products/stash/0.8.3/concepts/crds/backupconfiguration/)
- [RestoreSession](https://appscode.com/products/stash/0.8.3/concepts/crds/restoresession/)

To keep things isolated, we are going to use a separate namespace called `demo` throughout this tutorial. Create `demo` namespace if you haven't created yet.

```console
$ kubectl create ns demo
namespace/demo created
```

> Note: YAML files used in this tutorial are stored [here](https://github.com/stashed/mongodb/tree/master/docs/examples).

## Install MongoDB Catalog for Stash

Stash uses a `Function-Task` model to backup databases. We have to install MongoDB catalogs (`stash-mongodb`) for Stash. This catalog creates necessary `Function` and `Task` definitions to backup/restore MongoDB databases.

You can install the catalog either as a helm chart or you can create only the YAMLs of the respective resources.

<ul class="nav nav-tabs" id="installerTab" role="tablist">
  <li class="nav-item">
    <a class="nav-link" id="helm-tab" data-toggle="tab" href="#helm" role="tab" aria-controls="helm" aria-selected="false">Helm</a>
  </li>
  <li class="nav-item">
    <a class="nav-link active" id="script-tab" data-toggle="tab" href="#script" role="tab" aria-controls="script" aria-selected="true">Script</a>
  </li>
</ul>
<div class="tab-content" id="installerTabContent">
 <!-- ------------ Helm Tab Begins----------- -->
  <div class="tab-pane fade" id="helm" role="tabpanel" aria-labelledby="helm-tab">

### Install as chart release

Run the following script to install `stash-mongodb` catalog as a Helm chart.

```console
curl -fsSL https://github.com/stashed/catalog/raw/master/deploy/chart.sh | bash -s -- --catalog=stash-mongodb
```

</div>
<!-- ------------ Helm Tab Ends----------- -->

<!-- ------------ Script Tab Begins----------- -->
<div class="tab-pane fade show active" id="script" role="tabpanel" aria-labelledby="script-tab">

### Install only YAMLs

Run the following script to install `stash-mongodb` catalog as Kubernetes YAMLs.

```console
curl -fsSL https://github.com/stashed/catalog/raw/master/deploy/script.sh | bash -s -- --catalog=stash-mongodb
```

</div>
<!-- ------------ Script Tab Ends----------- -->
</div>

Once installed, this will create `mongodb-backup-*` and `mongodb-restore-*` Functions for all supported MongoDB versions. To verify, run the following command:

```console
$ kubectl get functions.stash.appscode.com
NAME                    AGE
mongodb-backup-4.1      20s
mongodb-backup-4.0      20s
mongodb-backup-3.6      19s
mongodb-backup-3.4      20s
mongodb-restore-4.1     20s
mongodb-restore-4.0     20s
mongodb-restore-3.6     19s
mongodb-restore-3.4     20s
pvc-backup              7h6m
pvc-restore             7h6m
update-status           7h6m
```

Also, verify that the necessary `Task` have been created.

```console
$ kubectl get tasks.stash.appscode.com
NAME                    AGE
mongodb-backup-4.1      2m7s
mongodb-backup-4.0      2m7s
mongodb-backup-3.6      2m6s
mongodb-backup-3.4      2m7s
mongodb-restore-4.1     2m7s
mongodb-restore-4.0     2m7s
mongodb-restore-3.6     2m6s
mongodb-restore-3.4     2m7s
pvc-backup              7h7m
pvc-restore             7h7m
```

Now, Stash is ready to backup MongoDB database.

## Backup MongoDB

This section will demonstrate how to backup MongoDB database. Here, we are going to deploy a MongoDB database using KubeDB. Then, we are going to backup this database into a GCS bucket. Finally, we are going to restore the backed up data into another MongoDB database.

### Deploy Sample MongoDB Database

Let's deploy a sample MongoDB database and insert some data into it.

**Create MongoDB CRD:**

Below is the YAML of a sample MongoDB crd that we are going to create for this tutorial:

```yaml
apiVersion: kubedb.com/v1alpha1
kind: MongoDB
metadata:
  name: sample-mongodb
  namespace: demo
spec:
  version: "3.6-v4"
  storageType: Durable
  storage:
    storageClassName: "standard"
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi
  terminationPolicy: WipeOut
```

Create the above `MongoDB` crd,

```console
$ kubectl apply -f ./docs/examples/backup/standalone/mongodb.yaml
mongodb.kubedb.com/sample-mongodb created
```

KubeDB will deploy a MongoDB database according to the above specification. It will also create the necessary secrets and services to access the database.

Let's check if the database is ready to use,

```console
$ kubectl get mg -n demo sample-mongodb
NAME             VERSION   STATUS    AGE
sample-mongodb   3.6-v4    Running   2m9s
```

The database is `Running`. Verify that KubeDB has created a Secret and a Service for this database using the following commands,

```console
$ kubectl get secret -n demo -l=kubedb.com/name=sample-mongodb
NAME                  TYPE     DATA   AGE
sample-mongodb-auth   Opaque   2      2m28s

$ kubectl get service -n demo -l=kubedb.com/name=sample-mongodb
NAME                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)     AGE
sample-mongodb       ClusterIP   10.107.58.222   <none>        27017/TCP   2m48s
sample-mongodb-gvr   ClusterIP   None            <none>        27017/TCP   2m48s
```

Here, we have to use service `sample-mongodb` and secret `sample-mongodb-auth` to connect with the database. KubeDB creates an [AppBinding](https://appscode.com/products/stash/0.8.3/concepts/crds/appbinding/) crd that holds the necessary information to connect with the database.

**Verify AppBinding:**

Verify that the `AppBinding` has been created successfully using the following command,

```console
$ kubectl get appbindings -n demo
NAME              AGE
sample-mongodb   20m
```

Let's check the YAML of the above `AppBinding`,

```console
$ kubectl get appbindings -n demo sample-mongodb -o yaml
```

```yaml
apiVersion: appcatalog.appscode.com/v1alpha1
kind: AppBinding
metadata:
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: sample-mongodb
    app.kubernetes.io/managed-by: kubedb.com
    app.kubernetes.io/name: mongodb
    app.kubernetes.io/version: 3.6-v4
    kubedb.com/kind: MongoDB
    kubedb.com/name: sample-mongodb
  name: sample-mongodb
  namespace: demo
spec:
  clientConfig:
    service:
      name: sample-mongodb
      port: 27017
      scheme: mongodb
  secret:
    name: sample-mongodb-auth
  type: kubedb.com/mongodb
```

Stash uses the `AppBinding` crd to connect with the target database. It requires the following two fields to set in AppBinding's `Spec` section.

- `spec.clientConfig.service.name` specifies the name of the service that connects to the database.
- `spec.secret` specifies the name of the secret that holds necessary credentials to access the database.
- `spec.type` specifies the types of the app that this AppBinding is pointing to. KubeDB generated AppBinding follows the following format: `<app group>/<app resource type>`.

### AppBinding for SSL

If `SSLMode` of the MongoDB server is either of `requireSSL` or `preferSSL`, you can provide ssl connection informatin through AppBinding Spces.

User need to provide the following fields in case of SSL is enabled,

- `spec.clientConfig.caBundle` specifies the CA certificate that is used in [`--sslCAFile`](https://docs.mongodb.com/manual/reference/program/mongod/index.html#cmdoption-mongod-sslcafile) flag of `mongod`.
- `spec.secret` specifies the name of the secret that holds `client.pem` file. Follow the [mongodb official doc](https://docs.mongodb.com/manual/tutorial/configure-x509-client-authentication/) to learn how to create `client.pem` and add the subject of `client.pem` as user (with appropriate roles) to mongodb server.

**KubeDB does these automatically**. It has added the subject of `client.pem` in the mongodb server with `root` role. So, user can just use the appbinding that is created by KubeDB without doing any hurdle! See the [MongoDB with TLS/SSL (Transport Encryption)](https://github.com/kubedb/docs/blob/master/docs/guides/mongodb/tls-ssl-encryption/tls-ssl-encryption.md) guide to learn about the ssl options in mongodb in details.

So, in KubeDB, the following `CRD` deployes a mongodb replicaset where ssl is enabled (`requireSSL` sslmode),

```yaml
apiVersion: kubedb.com/v1alpha1
kind: MongoDB
metadata:
  name: sample-mongodb
  namespace: demo
spec:
  version: "3.6-v4"
  storageType: Durable
  storage:
    storageClassName: "standard"
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi
  terminationPolicy: WipeOut
  sslMode: requireSSL
```

After the deploy is done, kubedb will create a appbinding that will look like:

```yaml
apiVersion: appcatalog.appscode.com/v1alpha1
kind: AppBinding
metadata:
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: sample-mongodb
    app.kubernetes.io/managed-by: kubedb.com
    app.kubernetes.io/name: mongodb
    app.kubernetes.io/version: 3.6-v4
    kubedb.com/kind: MongoDB
    kubedb.com/name: sample-mongodb
  name: sample-mongodb
  namespace: demo
spec:
  clientConfig:
    caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM0RENDQWNpZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFoTVJJd0VBWURWUVFLRXdsTGRXSmwKWkdJNlEwRXhDekFKQmdOVkJBTVRBa05CTUI0WERURTVNRGd3TVRFd01Ua3lPVm9YRFRJNU1EY3lPVEV3TVRreQpPVm93SVRFU01CQUdBMVVFQ2hNSlMzVmlaV1JpT2tOQk1Rc3dDUVlEVlFRREV3SkRRVENDQVNJd0RRWUpLb1pJCmh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTTQvWXpwbGhNTWlkUmREZEM0ejcvOXdkNWpZaitKeVl3d0EKQTJncktvYkxkQlB1YTNWNFB2TjJOOHNCaXArOUZycFFPVXkzOFpBdmZ4a1V1YkZOUVNoS3JkNnlGbWZRQjBhbApHOHB3dkcwblJoWEdWTHI3REVaaysrSjhZQWZwbzBlOHR6K29zZDVRMkpjN3JiRlFVbVFDYzJzc0ZXcGJSdy93CjFmeWhlaldTSjVnUnBYUG96ZHhxRkloTm9sbVgrQiswWVdrVHJ0QmJlcUZibnhkTm9oVmJDYzJtaHZOdnNoMHUKdWIvY1h1anhQR3ljNzZLYTEyZVhUS3FWTm9Jczg1TkdEbzlaSXBWZUhkRG1ld1ZQZVROcFlxWE9zMTRSOTNHWgozb0FoWW5JbG5veGFrOXEreE1Td01Vd0hwL0JyL0dRSkdGRlEyVDdSWWNMUS9HclYrdGtDQXdFQUFhTWpNQ0V3CkRnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCL3dRRk1BTUJBZjh3RFFZSktvWklodmNOQVFFTEJRQUQKZ2dFQkFLMXphNHQ5NjZBejYwZmJyQWh2bW5ZTXlPNUIvVWxaT01RcGhqRWRMOVBHekpSMG1uY1FOeWNoTFNqQwpkeDFaRG1iME9iSzh3WUgrbisyaW9DWTdiRFZBSjZTaHU3SkhBeDl4NGRkOG1HR0pCN1NUMGlxL3RJbGJmK2J0ClBNUnBEYmJ5YUZTVnpacXJvdTFJNkZycUwvQXVhTThzTUg5KzYzOW5zVS9XQWtIZWVVN0hhOWdpZXNwR0QrdGoKcUowOHBXcDB5Wndaa1plK3RyVUR6QmI1Um9VaWlMTGkxZXlKN0oyZmtvNk1OcXY1UkF2R2g2Y1ROcEtCbUdKQgpvaTUvSTgyQ1ovaGVYSXdpUkFrZ2NEbXpVRW1kaEk2OUZCMlQxVTNNVGNaaWtaSnRUdW13SXpFbWFZK3NySVdtCkZCdU1MMmdCRUhibEt3NFBjdmY3dWcvOGlHRT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    service:
      name: sample-mongodb
      port: 27017
      scheme: mongodb
  secret:
    name: sample-mongodb-cert
  type: kubedb.com/mongodb
```

Here, `sample-mongodb-cert` contains few required certificates, and one of them is `client.pem` which is required to backup/restore ssl enabled mongodb server using stash-mongodb.

**Creating AppBinding Manually:**

If you deploy MongoDB database without KubeDB, you have to create the AppBinding crd manually in the same namespace as the service and secret of the database.

The following YAML shows a minimal AppBinding specification that you have to create if you deploy MongoDB database without KubeDB.

```yaml
apiVersion: appcatalog.appscode.com/v1alpha1
kind: AppBinding
metadata:
  name: my-custom-appbinding
  namespace: my-database-namespace
spec:
  clientConfig:
    service:
      name: my-database-service
      port: 27017
  secret:
    name: my-database-credentials-secret
  # type field is optional. you can keep it empty.
  # if you keep it emtpty then the value of TARGET_APP_RESOURCE variable
  # will be set to "appbinding" during auto-backup.
  type: mongodb
```

**Insert Sample Data:**

Now, we are going to exec into the database pod and create some sample data. At first, find out the database pod using the following command,

```console
$ kubectl get pods -n demo --selector="kubedb.com/name=sample-mongodb"
NAME               READY   STATUS    RESTARTS   AGE
sample-mongodb-0   1/1     Running   0          12m
```

Now, let's exec into the pod and create a table,

```console
$ kubectl get secrets -n demo sample-mongodb-auth -o jsonpath='{.data.\username}' | base64 -d
root

$ kubectl get secrets -n demo sample-mongodb-auth -o jsonpath='{.data.\password}' | base64 -d
Tv1pSiLjGqZ9W4jE

$ kubectl exec -it -n demo sample-mongodb-0 bash

mongodb@sample-mongodb-0:/$ mongo admin -u root -p Tv1pSiLjGqZ9W4jE

> show dbs
admin  0.000GB
local  0.000GB
mydb   0.000GB

> show users
{
    "_id" : "admin.root",
    "user" : "root",
    "db" : "admin",
    "roles" : [
        {
            "role" : "root",
            "db" : "admin"
        }
    ]
}

> use newdb
switched to db newdb

> db.movie.insert({"name":"batman"});
WriteResult({ "nInserted" : 1 })

> db.movie.find().pretty()
{ "_id" : ObjectId("5d19d1cdc93d828f44e37735"), "name" : "batman" }

> exit
bye
```

Now, we are ready to backup this sample database.

### Prepare Backend

We are going to store our backed up data into a GCS bucket. At first, we need to create a secret with GCS credentials then we need to create a `Repository` crd. If you want to use a different backend, please read the respective backend configuration doc from [here](https://appscode.com/products/stash/0.8.3/guides/backends/overview/).

**Create Storage Secret:**

Let's create a secret called `gcs-secret` with access credentials to our desired GCS bucket,

```console
$ echo -n 'changeit' > RESTIC_PASSWORD
$ echo -n '<your-project-id>' > GOOGLE_PROJECT_ID
$ cat downloaded-sa-json.key > GOOGLE_SERVICE_ACCOUNT_JSON_KEY
$ kubectl create secret generic -n demo gcs-secret \
    --from-file=./RESTIC_PASSWORD \
    --from-file=./GOOGLE_PROJECT_ID \
    --from-file=./GOOGLE_SERVICE_ACCOUNT_JSON_KEY
secret/gcs-secret created
```

**Create Repository:**

Now, crete a `Respository` using this secret. Below is the YAML of Repository crd we are going to create,

```yaml
apiVersion: stash.appscode.com/v1alpha1
kind: Repository
metadata:
  name: gcs-repo
  namespace: demo
spec:
  backend:
    gcs:
      bucket: appscode-qa
      prefix: /demo/mongodb/sample-mongodb
    storageSecretName: gcs-secret
```

Let's create the `Repository` we have shown above,

```console
$ kubectl apply -f ./docs/examples/backup/standalone/repository.yaml
repository.stash.appscode.com/gcs-repo created
```

Now, we are ready to backup our database to our desired backend.

### Backup

We have to create a `BackupConfiguration` targeting respective AppBinding crd of our desired database. Then Stash will create a CronJob to periodically backup the database.

**Create BackupConfiguration:**

Below is the YAML for `BackupConfiguration` crd to backup the `sample-mongodb` database we have deployed earlier.,

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupConfiguration
metadata:
  name: sample-mongodb-backup
  namespace: demo
spec:
  schedule: "*/5 * * * *"
  task:
    name: mongodb-backup-3.6
  repository:
    name: gcs-repo
  target:
    ref:
      apiVersion: appcatalog.appscode.com/v1alpha1
      kind: AppBinding
      name: sample-mongodb
  retentionPolicy:
    keepLast: 5
    prune: true
```

Here,

- `spec.schedule` specifies that we want to backup the database at 5 minutes interval.
- `spec.task.name` specifies the name of the task crd that specifies the necessary Function and their execution order to backup a MongoDB database.
- `spec.target.ref` refers to the `AppBinding` crd that was created for `sample-mongodb` database.

Let's create the `BackupConfiguration` crd we have shown above,

```console
$ kubectl apply -f ./docs/examples/backup/standalone/backupconfiguration.yaml
backupconfiguration.stash.appscode.com/sample-mongodb-backup created
```

**Verify CronJob:**

If everything goes well, Stash will create a CronJob with the schedule specified in `spec.schedule` field of `BackupConfiguration` crd.

Verify that the CronJob has been created using the following command,

```console
$ kubectl get cronjob -n demo
NAME                    SCHEDULE      SUSPEND   ACTIVE   LAST SCHEDULE   AGE
sample-mongodb-backup   */5 * * * *   False     0        <none>          61s
```

**Wait for BackupSession:**

The `sample-mongodb-backup` CronJob will trigger a backup on each schedule by creating a `BackpSession` crd.

Wait for the next schedule. Run the following command to watch `BackupSession` crd,

```console
$ kubectl get backupsession -n demo -w
NAME                               BACKUPCONFIGURATION      PHASE       AGE
sample-mongodb-backup-1561974001   sample-mongodb-backup   Running     5m19s
sample-mongodb-backup-1561974001   sample-mongodb-backup   Succeeded   5m45s
```

We can see above that the backup session has succeeded. Now, we are going to verify that the backed up data has been stored in the backend.

**Verify Backup:**

Once a backup is complete, Stash will update the respective `Repository` crd to reflect the backup. Check that the repository `gcs-repo` has been updated by the following command,

```console
$ kubectl get repository -n demo gcs-repo
NAME       INTEGRITY   SIZE        SNAPSHOT-COUNT   LAST-SUCCESSFUL-BACKUP   AGE
gcs-repo   true        1.611 KiB   1                33s                      33m
```

Now, if we navigate to the GCS bucket, we are going to see backed up data has been stored in `demo/mongodb/sample-mongodb` directory as specified by `spec.backend.gcs.prefix` field of Repository crd.

> Note: Stash keeps all the backed up data encrypted. So, data in the backend will not make any sense until they are decrypted.

## Restore MongoDB

In this section, we are going to restore the database from the backup we have taken in the previous section. We are going to deploy a new database and initialize it from the backup.

**Stop Taking Backup of the Old Database:**

At first, let's stop taking any further backup of the old database so that no backup is taken during restore process. We are going to pause the `BackupConfiguration` crd that we had created to backup the `sample-mongodb` database. Then, Stash will stop taking any further backup for this database.

Let's pause the `sample-mongodb-backup` BackupConfiguration,

```console
$ kubectl patch backupconfiguration -n demo sample-mongodb-backup --type="merge" --patch='{"spec": {"paused": true}}'
backupconfiguration.stash.appscode.com/sample-mongodb-backup patched
```

Now, wait for a moment. Stash will pause the BackupConfiguration. Verify that the BackupConfiguration  has been paused,

```console
$ kubectl get backupconfiguration -n demo sample-mongodb-backup
NAME                   TASK                      SCHEDULE      PAUSED   AGE
sample-mongodb-backup  mongodb-backup-3.6        */5 * * * *   true     26m
```

Notice the `PAUSED` column. Value `true` for this field means that the BackupConfiguration has been paused.

**Deploy Restored Database:**

Now, we have to deploy the restored database similarly as we have deployed the original `sample-psotgres` database. However, this time there will be the following differences:

- We have to use the same secret that was used in the original database. We are going to specify it using `spec.databaseSecret` field.
- We have to specify `spec.init` section to tell KubeDB that we are going to use Stash to initialize this database from backup. KubeDB will keep the database phase to `Initializing` until Stash finishes its initialization.

Below is the YAML for `MongoDB` crd we are going deploy to initialize from backup,

```yaml
apiVersion: kubedb.com/v1alpha1
kind: MongoDB
metadata:
  name: restored-mongodb
  namespace: demo
spec:
  version: "3.6"
  storageType: Durable
  databaseSecret:
    secretName: sample-mongodb-auth # use same secret as original the database
  storage:
    storageClassName: "standard"
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi
  init:
    stashRestoreSession:
      name: sample-mongodb-restore
  terminationPolicy: WipeOut
```

Here,

- `spec.init.stashRestoreSession.name` specifies the `RestoreSession` crd name that we are going to use to restore this database.

Let's create the above database,

```console
$ kubectl apply -f ./docs/examples/restore/standalone/restored-mongodb.yaml
mongodb.kubedb.com/restored-mongodb created
```

If you check the database status, you will see it is stuck in `Initializing` state.

```console
$ kubectl get mg -n demo restored-mongodb
NAME               VERSION   STATUS         AGE
restored-mongodb   3.6-v4    Initializing   17s
```

**Create RestoreSession:**

Now, we need to create a `RestoreSession` crd pointing to the AppBinding for this restored database.

Check AppBinding has been created for the `restored-mongodb` database using the following command,

```console
$ kubectl get appbindings -n demo restored-mongodb
NAME               AGE
restored-mongodb   29s
```

> If you are not using KubeDB to deploy database, create the AppBinding manually.

Below is the YAML for the `RestoreSession` crd that we are going to create to restore backed up data into `restored-mongodb` database.

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: RestoreSession
metadata:
  name: sample-mongodb-restore
  namespace: demo
  labels:
    kubedb.com/kind: MongoDB
spec:
  task:
    name: mongodb-restore-3.6
  repository:
    name: gcs-repo
  target:
    ref:
      apiVersion: appcatalog.appscode.com/v1alpha1
      kind: AppBinding
      name: restored-mongodb
  rules:
    - snapshots: [latest]
```

Here,

- `metadata.labels` specifies a `kubedb.com/kind: MongoDB` label that is used by KubeDB to watch this `RestoreSession`.
- `spec.task.name` specifies the name of the `Task` crd that specifies the Functions and their execution order to restore a MongoDB database.
- `spec.repository.name` specifies the `Repository` crd that holds the backend information where our backed up data has been stored.
- `spec.target.ref` refers to the AppBinding crd for the `restored-mongodb` database.
- `spec.rules` specifies that we are restoring from the latest backup snapshot of the database.

> **Warning:** Label `kubedb.com/kind: MongoDB` is mandatory if you are uisng KubeDB to deploy the database. Otherwise, the database will be stuck in `Initializing` state.

Let's create the `RestoreSession` crd we have shown above,

```console
$ kubectl apply -f ./docs/examples/restore/standalone/restoresession.yaml
restoresession.stash.appscode.com/sample-mongodb-restore created
```

Once, you have created the `RestoreSession` crd, Stash will create a job to restore. We can watch the `RestoreSession` phase to check if the restore process is succeeded or not.

Run the following command to watch `RestoreSession` phase,

```console
$ kubectl get restoresession -n demo sample-mongodb-restore -w
NAME                     REPOSITORY-NAME   PHASE       AGE
sample-mongodb-restore   gcs-repo          Running     5s
sample-mongodb-restore   gcs-repo          Succeeded   43s
```

So, we can see from the output of the above command that the restore process succeeded.

**Verify Restored Data:**

In this section, we are going to verify that the desired data has been restored successfully. We are going to connect to the database and check whether the table we had created in the original database is restored or not.

At first, check if the database has gone into `Running` state by the following command,

```console
$ kubectl get mg -n demo restored-mongodb
NAME               VERSION   STATUS    AGE
restored-mongodb   3.6-v4    Running   105m
```

Now, find out the database pod by the following command,

```console
$ kubectl get pods -n demo --selector="kubedb.com/name=restored-mongodb"
NAME                 READY   STATUS    RESTARTS   AGE
restored-mongodb-0   1/1     Running   0          106m
```

Now, exec into the database pod and list available tables,

```console
$ kubectl get secrets -n demo sample-mongodb-auth -o jsonpath='{.data.\username}' | base64 -d
root

$ kubectl get secrets -n demo sample-mongodb-auth -o jsonpath='{.data.\password}' | base64 -d
Tv1pSiLjGqZ9W4jE

$ kubectl exec -it -n demo restored-mongodb-0 bash

mongodb@restored-mongodb-0:/$ mongo admin -u root -p Tv1pSiLjGqZ9W4jE

> show dbs
admin   0.000GB
config  0.000GB
local   0.000GB
newdb   0.000GB

> show users
{
    "_id" : "admin.root",
    "user" : "root",
    "db" : "admin",
    "roles" : [
        {
            "role" : "root",
            "db" : "admin"
        }
    ]
}

> use newdb
switched to db newdb

> db.movie.find().pretty()
{ "_id" : ObjectId("5d19d1cdc93d828f44e37735"), "name" : "batman" }

> exit
bye
```

So, from the above output, we can see the database `newdb` that we had created in the original database `sample-mongodb` is restored in the restored database `restored-mongodb`.

## Cleanup

To cleanup the Kubernetes resources created by this tutorial, run:

```console
kubectl delete restoresession -n demo sample-mongodb-restore
kubectl delete backupconfiguration -n demo sample-mongodb-backup
kubectl delete mg -n demo restored-mongodb
kubectl delete mg -n demo sample-mongodb
```

To cleanup the MongoDB catalogs that we had created earlier, run the following:

<ul class="nav nav-tabs" id="uninstallerTab" role="tablist">
  <li class="nav-item">
    <a class="nav-link" id="helm-uninstaller-tab" data-toggle="tab" href="#helm-uninstaller" role="tab" aria-controls="helm-uninstaller" aria-selected="false">Helm</a>
  </li>
  <li class="nav-item">
    <a class="nav-link active" id="script-uninstaller-tab" data-toggle="tab" href="#script-uninstaller" role="tab" aria-controls="script-uninstaller" aria-selected="true">Script</a>
  </li>
</ul>
<div class="tab-content" id="uninstallerTabContent">
 <!-- ------------ Helm Tab Begins----------- -->
  <div class="tab-pane fade" id="helm-uninstaller" role="tabpanel" aria-labelledby="helm-uninstaller-tab">

### Uninstall  `stash-mongdb-*` charts

Run the following script to uninstall `stash-mongodb` catalogs that was installed as a Helm chart.

```console
curl -fsSL https://github.com/stashed/catalog/raw/master/deploy/chart.sh | bash -s -- --uninstall --catalog=stash-mongodb
```

</div>
<!-- ------------ Helm Tab Ends----------- -->

<!-- ------------ Script Tab Begins----------- -->
<div class="tab-pane fade show active" id="script-uninstaller" role="tabpanel" aria-labelledby="script-uninstaller-tab">

### Uninstall `stash-mongodb` catalog YAMLs

Run the following script to uninstall `stash-mongodb` catalog that was installed as Kubernetes YAMLs.

```console
curl -fsSL https://github.com/stashed/catalog/raw/master/deploy/script.sh | bash -s -- --uninstall --catalog=stash-mongodb
```

</div>
<!-- ------------ Script Tab Ends----------- -->
</div>
