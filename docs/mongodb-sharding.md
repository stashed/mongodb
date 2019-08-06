---
title: Backup MongoDB Sharded Cluster | Stash
description: Backup MongoDB Sharded Cluster using Stash
menu:
  product_stash_0.8.3:
    identifier: sharded-mongodb
    name: MongoDB
    parent: database
    weight: 30
product_name: stash
menu_name: product_stash_0.8.3
section_menu_id: guides
---

# Backup and Restore MongoDB Sharded Clusters using Stash

Stash 0.9.0+ supports taking [backup](https://docs.mongodb.com/manual/tutorial/backup-sharded-cluster-with-database-dumps/) and [restores](https://docs.mongodb.com/manual/tutorial/restore-sharded-cluster/) MongoDB Sharded clusters in ["idiomatic" way](https://docs.mongodb.com/manual/administration/backup-sharded-clusters/). This guide will show you how you can backup and restore your MongoDB Sharded clusters with Stash.

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

## Backup Sharded MongoDB Cluster

This section will demonstrate how to backup MongoDB cluster. We are going to use [KubeDB](https://kubedb.com) to deploy a sample database. Then, we are going to backup this database into a GCS bucket. Finally, we are going to restore the backed up data into another MongoDB cluster.

### Deploy Sample MongoDB Sharding

Let's deploy a sample MongoDB Sharding database and insert some data into it.

**Create MongoDB CRD:**

Below is the YAML of a sample MongoDB crd that we are going to create for this tutorial:

```yaml
apiVersion: kubedb.com/v1alpha1
kind: MongoDB
metadata:
  name: sample-mgo-sh
  namespace: demo
spec:
  version: 3.6-v4
  shardTopology:
    configServer:
      replicas: 3
      storage:
        resources:
          requests:
            storage: 1Gi
        storageClassName: standard
    mongos:
      replicas: 2
      strategy:
        type: RollingUpdate
    shard:
      replicas: 3
      shards: 3
      storage:
        resources:
          requests:
            storage: 1Gi
        storageClassName: standard
  terminationPolicy: WipeOut
```

Create the above `MongoDB` crd,

```console
$ kubectl apply -f ./docs/examples/backup/sharding/mongodb-sharding.yaml
mongodb.kubedb.com/sample-mgo-sh created
```

KubeDB will deploy a MongoDB database according to the above specification. It will also create the necessary secrets and services to access the database.

Let's check if the database is ready to use,

```console
$ kubectl get mg -n demo sample-mgo-sh
NAME            VERSION   STATUS    AGE
sample-mgo-sh   3.6-v4    Running   35m
```

The database is `Running`. Verify that KubeDB has created a Secret and a Service for this database using the following commands,

```console
$ kubectl get secret -n demo -l=kubedb.com/name=sample-mgo-sh
NAME                 TYPE     DATA   AGE
sample-mgo-sh-auth   Opaque   2      36m
sample-mgo-sh-cert   Opaque   4      36m

$ kubectl get service -n demo -l=kubedb.com/name=sample-mgo-sh
NAME                          TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)     AGE
sample-mgo-sh                 ClusterIP   10.107.11.117   <none>        27017/TCP   36m
sample-mgo-sh-configsvr-gvr   ClusterIP   None            <none>        27017/TCP   36m
sample-mgo-sh-shard0-gvr      ClusterIP   None            <none>        27017/TCP   36m
sample-mgo-sh-shard1-gvr      ClusterIP   None            <none>        27017/TCP   36m
sample-mgo-sh-shard2-gvr      ClusterIP   None            <none>        27017/TCP   36m
```

KubeDB creates an [AppBinding](https://appscode.com/products/stash/0.8.3/concepts/crds/appbinding/) crd that holds the necessary information to connect with the database.

**Verify AppBinding:**

Verify that the `AppBinding` has been created successfully using the following command,

```console
$ kubectl get appbindings -n demo
NAME            AGE
sample-mgo-sh   30m
```

Let's check the YAML of the above `AppBinding`,

```console
$ kubectl get appbindings -n demo sample-mgo-sh -o yaml
```

```yaml
apiVersion: appcatalog.appscode.com/v1alpha1
kind: AppBinding
metadata:
  creationTimestamp: "2019-07-18T11:18:24Z"
  generation: 1
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: sample-mgo-sh
    app.kubernetes.io/managed-by: kubedb.com
    app.kubernetes.io/name: mongodb
    app.kubernetes.io/version: 3.6-v4
    kubedb.com/kind: MongoDB
    kubedb.com/name: sample-mgo-sh
  name: sample-mgo-sh
  namespace: demo
  ownerReferences:
    - apiVersion: kubedb.com/v1alpha1
      blockOwnerDeletion: false
      kind: MongoDB
      name: sample-mgo-sh
      uid: d1e72f6d-a94c-11e9-acf7-42010a8000dc
  resourceVersion: "870217"
  selfLink: /apis/appcatalog.appscode.com/v1alpha1/namespaces/demo/appbindings/sample-mgo-sh
  uid: c247c3bd-a94d-11e9-acf7-42010a8000dc
spec:
  clientConfig:
    service:
      name: sample-mgo-sh
      port: 27017
      scheme: mongodb
  parameters:
    configServer: cnfRepSet/sample-mgo-sh-configsvr-0.sample-mgo-sh-configsvr-gvr.demo.svc:27017,sample-mgo-sh-configsvr-1.sample-mgo-sh-configsvr-gvr.demo.svc:27017,sample-mgo-sh-configsvr-2.sample-mgo-sh-configsvr-gvr.demo.svc:27017
    replicaSets:
      host-0: shard0/sample-mgo-sh-shard0-0.sample-mgo-sh-shard0-gvr.demo.svc:27017,sample-mgo-sh-shard0-1.sample-mgo-sh-shard0-gvr.demo.svc:27017,sample-mgo-sh-shard0-2.sample-mgo-sh-shard0-gvr.demo.svc:27017
      host-1: shard1/sample-mgo-sh-shard1-0.sample-mgo-sh-shard1-gvr.demo.svc:27017,sample-mgo-sh-shard1-1.sample-mgo-sh-shard1-gvr.demo.svc:27017,sample-mgo-sh-shard1-2.sample-mgo-sh-shard1-gvr.demo.svc:27017
      host-2: shard2/sample-mgo-sh-shard2-0.sample-mgo-sh-shard2-gvr.demo.svc:27017,sample-mgo-sh-shard2-1.sample-mgo-sh-shard2-gvr.demo.svc:27017,sample-mgo-sh-shard2-2.sample-mgo-sh-shard2-gvr.demo.svc:27017
  secret:
    name: sample-mgo-sh-auth
  type: kubedb.com/mongodb
```

Stash uses the `AppBinding` crd to connect with the target database. It requires the following two fields to set in AppBinding's `Spec` section.

- `spec.clientConfig.service.name` specifies the name of the service that connects to the database.
- `spec.secret` specifies the name of the secret that holds necessary credentials to access the database.
- `spec.parameters.configServer` specifies the dsn of config server of mongodb sharding. The dsn includes the port no too.
- `spec.parameters.replicaSets` contains the dsn of each replicaset of sharding. The DSNs are in key-value pair, where the keys are host-0, host-1 etc, and the values are DSN of each replicaset. If there is no sharding but only one replicaset, then ReplicaSets field contains only one key-value pair where the key is host-0 and the value is dsn of that replicaset.
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
  name: sample-mgo-sh
  namespace: demo
spec:
  version: 3.6-v4
  shardTopology:
    configServer:
      replicas: 3
      storage:
        resources:
          requests:
            storage: 1Gi
        storageClassName: standard
    mongos:
      replicas: 2
      strategy:
        type: RollingUpdate
    shard:
      replicas: 3
      shards: 3
      storage:
        resources:
          requests:
            storage: 1Gi
        storageClassName: standard
  terminationPolicy: WipeOut
  clusterAuthMode: x509
  sslMode: requireSSL
```

After the deploy is done, kubedb will create a appbinding that will look like:

```yaml
apiVersion: appcatalog.appscode.com/v1alpha1
kind: AppBinding
metadata:
  creationTimestamp: "2019-07-18T11:18:24Z"
  generation: 1
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: sample-mgo-sh
    app.kubernetes.io/managed-by: kubedb.com
    app.kubernetes.io/name: mongodb
    app.kubernetes.io/version: 3.6-v4
    kubedb.com/kind: MongoDB
    kubedb.com/name: sample-mgo-sh
  name: sample-mgo-sh
  namespace: demo
  ownerReferences:
    - apiVersion: kubedb.com/v1alpha1
      blockOwnerDeletion: false
      kind: MongoDB
      name: sample-mgo-sh
      uid: d1e72f6d-a94c-11e9-acf7-42010a8000dc
  resourceVersion: "870217"
  selfLink: /apis/appcatalog.appscode.com/v1alpha1/namespaces/demo/appbindings/sample-mgo-sh
  uid: c247c3bd-a94d-11e9-acf7-42010a8000dc
spec:
  clientConfig:
    caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM0RENDQWNpZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFoTVJJd0VBWURWUVFLRXdsTGRXSmwKWkdJNlEwRXhDekFKQmdOVkJBTVRBa05CTUI0WERURTVNRGd3TVRFd01Ua3lPVm9YRFRJNU1EY3lPVEV3TVRreQpPVm93SVRFU01CQUdBMVVFQ2hNSlMzVmlaV1JpT2tOQk1Rc3dDUVlEVlFRREV3SkRRVENDQVNJd0RRWUpLb1pJCmh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTTQvWXpwbGhNTWlkUmREZEM0ejcvOXdkNWpZaitKeVl3d0EKQTJncktvYkxkQlB1YTNWNFB2TjJOOHNCaXArOUZycFFPVXkzOFpBdmZ4a1V1YkZOUVNoS3JkNnlGbWZRQjBhbApHOHB3dkcwblJoWEdWTHI3REVaaysrSjhZQWZwbzBlOHR6K29zZDVRMkpjN3JiRlFVbVFDYzJzc0ZXcGJSdy93CjFmeWhlaldTSjVnUnBYUG96ZHhxRkloTm9sbVgrQiswWVdrVHJ0QmJlcUZibnhkTm9oVmJDYzJtaHZOdnNoMHUKdWIvY1h1anhQR3ljNzZLYTEyZVhUS3FWTm9Jczg1TkdEbzlaSXBWZUhkRG1ld1ZQZVROcFlxWE9zMTRSOTNHWgozb0FoWW5JbG5veGFrOXEreE1Td01Vd0hwL0JyL0dRSkdGRlEyVDdSWWNMUS9HclYrdGtDQXdFQUFhTWpNQ0V3CkRnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCL3dRRk1BTUJBZjh3RFFZSktvWklodmNOQVFFTEJRQUQKZ2dFQkFLMXphNHQ5NjZBejYwZmJyQWh2bW5ZTXlPNUIvVWxaT01RcGhqRWRMOVBHekpSMG1uY1FOeWNoTFNqQwpkeDFaRG1iME9iSzh3WUgrbisyaW9DWTdiRFZBSjZTaHU3SkhBeDl4NGRkOG1HR0pCN1NUMGlxL3RJbGJmK2J0ClBNUnBEYmJ5YUZTVnpacXJvdTFJNkZycUwvQXVhTThzTUg5KzYzOW5zVS9XQWtIZWVVN0hhOWdpZXNwR0QrdGoKcUowOHBXcDB5Wndaa1plK3RyVUR6QmI1Um9VaWlMTGkxZXlKN0oyZmtvNk1OcXY1UkF2R2g2Y1ROcEtCbUdKQgpvaTUvSTgyQ1ovaGVYSXdpUkFrZ2NEbXpVRW1kaEk2OUZCMlQxVTNNVGNaaWtaSnRUdW13SXpFbWFZK3NySVdtCkZCdU1MMmdCRUhibEt3NFBjdmY3dWcvOGlHRT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    service:
      name: sample-mgo-sh
      port: 27017
      scheme: mongodb
  parameters:
    configServer: cnfRepSet/sample-mgo-sh-configsvr-0.sample-mgo-sh-configsvr-gvr.demo.svc:27017,sample-mgo-sh-configsvr-1.sample-mgo-sh-configsvr-gvr.demo.svc:27017,sample-mgo-sh-configsvr-2.sample-mgo-sh-configsvr-gvr.demo.svc:27017
    replicaSets:
      host-0: shard0/sample-mgo-sh-shard0-0.sample-mgo-sh-shard0-gvr.demo.svc:27017,sample-mgo-sh-shard0-1.sample-mgo-sh-shard0-gvr.demo.svc:27017,sample-mgo-sh-shard0-2.sample-mgo-sh-shard0-gvr.demo.svc:27017
      host-1: shard1/sample-mgo-sh-shard1-0.sample-mgo-sh-shard1-gvr.demo.svc:27017,sample-mgo-sh-shard1-1.sample-mgo-sh-shard1-gvr.demo.svc:27017,sample-mgo-sh-shard1-2.sample-mgo-sh-shard1-gvr.demo.svc:27017
      host-2: shard2/sample-mgo-sh-shard2-0.sample-mgo-sh-shard2-gvr.demo.svc:27017,sample-mgo-sh-shard2-1.sample-mgo-sh-shard2-gvr.demo.svc:27017,sample-mgo-sh-shard2-2.sample-mgo-sh-shard2-gvr.demo.svc:27017
  secret:
    name: sample-mgo-sh-cert
  type: kubedb.com/mongodb
```

Here, `sample-mgo-sh-cert` contains few required certificates, and one of them is `client.pem` which is required to backup/restore ssl enabled mongodb server using stash-mongodb.

**Creating AppBinding Manually:**

If you deploy MongoDB database without KubeDB, you have to create the AppBinding crd manually in the same namespace as the service and secret of the database.

**Insert Sample Data:**

Now, we are going to exec into the database pod and create some sample data. At first, find out the database pod using the following command,

```console
$ kubectl get pods -n demo --selector="mongodb.kubedb.com/node.mongos=sample-mgo-sh-mongos"
NAME                                   READY   STATUS    RESTARTS   AGE
sample-mgo-sh-mongos-9459cfc44-4jthd   1/1     Running   0          60m
sample-mgo-sh-mongos-9459cfc44-6d2st   1/1     Running   0          60m
```

Now, let's exec into the pod and create a table,

```console
$ kubectl get secrets -n demo sample-mgo-sh-auth -o jsonpath='{.data.\username}' | base64 -d
root

$ kubectl get secrets -n demo sample-mgo-sh-auth -o jsonpath='{.data.\password}' | base64 -d
JJPcMxNKJev0SzgX

$ kubectl exec -it -n demo sample-mgo-sh-mongos-9459cfc44-4jthd bash

mongodb@sample-mgo-sh-0:/$ mongo admin -u root -p JJPcMxNKJev0SzgX

mongos> show dbs
admin   0.000GB
config  0.001GB


mongos> show users
{
	"_id" : "admin.root",
	"userId" : UUID("b9a1551b-83cf-4ebb-852b-dd23c890f301"),
	"user" : "root",
	"db" : "admin",
	"roles" : [
		{
			"role" : "root",
			"db" : "admin"
		}
	]
}

mongos> use newdb
switched to db newdb

mongos> db.movie.insert({"name":"batman"});
WriteResult({ "nInserted" : 1 })

mongos> db.movie.find().pretty()
{ "_id" : ObjectId("5d3064bf144a1b8fda04cd4f"), "name" : "batman" }

mongos> exit
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
  name: gcs-repo-sharding
  namespace: demo
spec:
  backend:
    gcs:
      bucket: appscode-qa
      prefix: demo/mongodb/sample-mgo-sh
    storageSecretName: gcs-secret
```

Let's create the `Repository` we have shown above,

```console
$ kubectl apply -f ./docs/examples/backup/sharding/repository-sharding.yaml
repository.stash.appscode.com/gcs-repo-sharding created
```

Now, we are ready to backup our database to our desired backend.

### Backup MongoDB Sharding

We have to create a `BackupConfiguration` targeting respective AppBinding crd of our desired database. Then Stash will create a CronJob to periodically backup the database.

**Create BackupConfiguration:**

Below is the YAML for `BackupConfiguration` crd to backup the `sample-mgo-sh` database we have deployed earlier.,

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: BackupConfiguration
metadata:
  name: sample-mgo-sh-backup
  namespace: demo
spec:
  schedule: "*/5 * * * *"
  task:
    name: mg-backup-3.6
  repository:
    name: gcs-repo-sharding
  target:
    ref:
      apiVersion: appcatalog.appscode.com/v1alpha1
      kind: AppBinding
      name: sample-mgo-sh
  retentionPolicy:
    keepLast: 5
    prune: true
```

Here,

- `spec.schedule` specifies that we want to backup the database at 5 minutes interval.
- `spec.task.name` specifies the name of the task crd that specifies the necessary Function and their execution order to backup a MongoDB database.
- `spec.target.ref` refers to the `AppBinding` crd that was created for `sample-mgo-sh` database.

Let's create the `BackupConfiguration` crd we have shown above,

```console
$ kubectl apply -f ./docs/examples/backup/sharding/backupconfiguration-sharding.yaml
backupconfiguration.stash.appscode.com/sample-mgo-sh-backup created
```

**Verify CronJob:**

If everything goes well, Stash will create a CronJob with the schedule specified in `spec.schedule` field of `BackupConfiguration` crd.

Verify that the CronJob has been created using the following command,

```console
$ kubectl get cronjob -n demo
NAME                   SCHEDULE      SUSPEND   ACTIVE   LAST SCHEDULE   AGE
sample-mgo-sh-backup   */5 * * * *   False     0        <none>          13s
```

**Wait for BackupSession:**

The `sample-mgo-sh-backup` CronJob will trigger a backup on each schedule by creating a `BackpSession` crd.

Wait for the next schedule. Run the following command to watch `BackupSession` crd,

```console
$ kubectl get backupsession -n demo -w
NAME                               BACKUPCONFIGURATION      PHASE       AGE
sample-mgo-sh-backup-1563512707   sample-mgo-sh-backup   Running     5m19s
sample-mgo-sh-backup-1563512707   sample-mgo-sh-backup   Succeeded   5m45s
```

We can see above that the backup session has succeeded. Now, we are going to verify that the backed up data has been stored in the backend.

**Verify Backup:**

Once a backup is complete, Stash will update the respective `Repository` crd to reflect the backup. Check that the repository `gcs-repo-sharding` has been updated by the following command,

```console
$ kubectl get repository -n demo gcs-repo-sharding
NAME                INTEGRITY   SIZE         SNAPSHOT-COUNT   LAST-SUCCESSFUL-BACKUP   AGE
gcs-repo-sharding   true        66.453 KiB   12               1m                       20m
```

Now, if we navigate to the GCS bucket, we are going to see backed up data has been stored in `demo/mongodb/sample-mgo-sh` directory as specified by `spec.backend.gcs.prefix` field of Repository crd.

> Note: Stash keeps all the backed up data encrypted. So, data in the backend will not make any sense until they are decrypted.

## Restore MongoDB Sharding

In this section, we are going to restore the database from the backup we have taken in the previous section. We are going to deploy a new sharded database and initialize it from the backup.

**Stop Taking Backup of the Old Database:**

At first, let's stop taking any further backup of the old database so that no backup is taken during restore process. We are going to pause the `BackupConfiguration` crd that we had created to backup the `sample-mgo-sh` database. Then, Stash will stop taking any further backup for this database.

Let's pause the `sample-mgo-sh-backup` BackupConfiguration,

```console
$ kubectl patch backupconfiguration -n demo sample-mgo-sh-backup --type="merge" --patch='{"spec": {"paused": true}}'
backupconfiguration.stash.appscode.com/sample-mgo-sh-backup patched
```

Now, wait for a moment. Stash will pause the BackupConfiguration. Verify that the BackupConfiguration  has been paused,

```console
$ kubectl get backupconfiguration -n demo sample-mgo-sh-backup
NAME                  TASK                      SCHEDULE      PAUSED   AGE
sample-mgo-sh-backup  mongodb-backup-3.6        */5 * * * *   true     26m
```

Notice the `PAUSED` column. Value `true` for this field means that the BackupConfiguration has been paused.

**Deploy Restored Database:**

Now, we have to deploy the restored database similarly as we have deployed the original `sample-mgo-sh` database. However, this time there will be the following differences:

- We have to use the same secret that was used in the original database. We are going to specify it using `spec.databaseSecret` field.
- We have to specify `spec.init` section to tell KubeDB that we are going to use Stash to initialize this database from backup. KubeDB will keep the database phase to `Initializing` until Stash finishes its initialization.

Below is the YAML for `MongoDB` crd we are going deploy to initialize from backup,

```yaml
apiVersion: kubedb.com/v1alpha1
kind: MongoDB
metadata:
  name: restored-mgo-sh
  namespace: demo
spec:
  databaseSecret:
    secretName: sample-mgo-sh-auth
  version: 3.6-v4
  shardTopology:
    configServer:
      replicas: 3
      storage:
        resources:
          requests:
            storage: 1Gi
        storageClassName: standard
    mongos:
      replicas: 2
      strategy:
        type: RollingUpdate
    shard:
      replicas: 3
      shards: 3
      storage:
        resources:
          requests:
            storage: 1Gi
        storageClassName: standard
  init:
    stashRestoreSession:
      name: sample-mgo-sh-restore
  terminationPolicy: WipeOut
```

Here,

- `spec.init.stashRestoreSession.name` specifies the `RestoreSession` crd name that we are going to use to restore this database.

Let's create the above database,

```console
$ kubectl apply -f ./docs/examples/restore/sharding/restored-mongodb-sharding.yaml
mongodb.kubedb.com/restored-mgo-sh created
```

If you check the database status, you will see it is stuck in `Initializing` state.

```console
$ kubectl get mg -n demo restored-mgo-sh
NAME              VERSION   STATUS         AGE
restored-mgo-sh   3.6-v4    Initializing   48m
```

**Create RestoreSession:**

Now, we need to create a `RestoreSession` crd pointing to the AppBinding for this restored database.

Check AppBinding has been created for the `restored-mgo-sh` database using the following command,

```console
$ kubectl get appbindings -n demo restored-mgo-sh
NAME               AGE
restored-mgo-sh    29s
```

NB. The appbinding `restored-mgo-sh` also contains `spec.parametrs` field. the number of hosts in `spec.parameters.replicaSets` needs to be similar to the old appbinding. Otherwise, the sharding recover may not be accurate.

> If you are not using KubeDB to deploy database, create the AppBinding manually.

Below is the YAML for the `RestoreSession` crd that we are going to create to restore backed up data into `restored-mgo-sh` database.

```yaml
apiVersion: stash.appscode.com/v1beta1
kind: RestoreSession
metadata:
  name: sample-mgo-sh-restore
  namespace: demo
  labels:
    kubedb.com/kind: MongoDB
spec:
  task:
    name: mg-restore-3.6
  repository:
    name: gcs-repo-sharding
  target:
    ref:
      apiVersion: appcatalog.appscode.com/v1alpha1
      kind: AppBinding
      name: restored-mgo-sh
  rules:
    - snapshots: [latest]
```

Here,

- `metadata.labels` specifies a `kubedb.com/kind: MongoDB` label that is used by KubeDB to watch this `RestoreSession`.
- `spec.task.name` specifies the name of the `Task` crd that specifies the Functions and their execution order to restore a MongoDB database.
- `spec.repository.name` specifies the `Repository` crd that holds the backend information where our backed up data has been stored.
- `spec.target.ref` refers to the AppBinding crd for the `restored-mgo-sh` database.
- `spec.rules` specifies that we are restoring from the latest backup snapshot of the database.

> **Warning:** Label `kubedb.com/kind: MongoDB` is mandatory if you are uisng KubeDB to deploy the database. Otherwise, the database will be stuck in `Initializing` state.

Let's create the `RestoreSession` crd we have shown above,

```console
$ kubectl apply -f ./docs/examples/restore/sharding/restoresession-sharding.yaml
restoresession.stash.appscode.com/sample-mgo-sh-restore created
```

Once, you have created the `RestoreSession` crd, Stash will create a job to restore. We can watch the `RestoreSession` phase to check if the restore process is succeeded or not.

Run the following command to watch `RestoreSession` phase,

```console
$ kubectl get restoresession -n demo sample-mgo-sh-restore -w
NAME                    REPOSITORY-NAME      PHASE       AGE
sample-mgo-sh-restore   gcs-repo-sharding    Running     5s
sample-mgo-sh-restore   gcs-repo-sharding    Succeeded   43s
```

So, we can see from the output of the above command that the restore process succeeded.

**Verify Restored Data:**

In this section, we are going to verify that the desired data has been restored successfully. We are going to connect to `mongos` and check whether the table we had created in the original database is restored or not.

At first, check if the database has gone into `Running` state by the following command,

```console
$ kubectl get mg -n demo restored-mgo-sh
NAME              VERSION   STATUS    AGE
restored-mgo-sh   3.6-v4    Running   2h
```

Now, find out the `mongos` pod,

```console
$ kubectl get pods -n demo --selector="mongodb.kubedb.com/node.mongos=restored-mgo-sh-mongos"
NAME                                      READY   STATUS    RESTARTS   AGE
restored-mgo-sh-mongos-7bccd5d684-2z5xs   1/1     Running   0          169m
restored-mgo-sh-mongos-7bccd5d684-vvdxb   1/1     Running   0          169m
```

Now, exec into the database pod and list available tables,

```console
$ kubectl get secrets -n demo sample-mgo-sh-auth -o jsonpath='{.data.\username}' | base64 -d
root

$ kubectl get secrets -n demo sample-mgo-sh-auth -o jsonpath='{.data.\password}' | base64 -d
JJPcMxNKJev0SzgX

$ kubectl exec -it -n demo restored-mgo-sh-mongos-7bccd5d684-2z5xs bash

mongodb@restored-mgo-sh-0:/$ mongo admin -u root -p JJPcMxNKJev0SzgX

mongos> show dbs
admin   0.000GB
config  0.001GB
newdb   0.000GB


mongos> show users
{
	"_id" : "admin.root",
	"userId" : UUID("a57cb466-ec66-453b-b795-654169a0f035"),
	"user" : "root",
	"db" : "admin",
	"roles" : [
		{
			"role" : "root",
			"db" : "admin"
		}
	]
}

mongos> use newdb
switched to db newdb

mongos> db.movie.find().pretty()
{ "_id" : ObjectId("5d3064bf144a1b8fda04cd4f"), "name" : "batman" }

mongos> exit
bye
```

So, from the above output, we can see the database `newdb` that we had created in the original database `sample-mgo-sh` is restored in the restored database `restored-mgo-sh`.

## Backup MongoDB Sharded Cluster and Restore into a Standalone database

It is possible to take backup of a MongoDB Sharded Cluster and restore it into a standalone database, but user need to create the appbinding for this process.

### Backup a sharded cluster

Keep all the fields of appbinding that is explained earlier in this guide, except `spec.parameter`. Do not set `spec.parameter.configServer` and `spec.parameter.replicaSet`. By doing this, the job will use `spec.clientConfig.service.name` as host, which is `mongos` router DSN. So, the backup will treat this cluster as a standalone and will skip the [`idiomatic way` of taking backups of a sharded cluster](https://docs.mongodb.com/manual/tutorial/backup-sharded-cluster-with-database-dumps/). Then follow the rest of the procedure as described above.

```yaml
apiVersion: appcatalog.appscode.com/v1alpha1
kind: AppBinding
metadata:
  name: sample-mgo-sh-custom
  namespace: demo
spec:
  clientConfig:
    service:
      name: sample-mgo-sh
      port: 27017
      scheme: mongodb
  secret:
    name: sample-mgo-sh-auth
  type: kubedb.com/mongodb

---
apiVersion: stash.appscode.com/v1alpha1
kind: Repository
metadata:
  name: gcs-repo-custom
  namespace: demo
spec:
  backend:
    gcs:
      bucket: appscode-qa
      prefix: demo/mongodb/sample-mgo-sh/standalone
    storageSecretName: gcs-secret

---
apiVersion: stash.appscode.com/v1beta1
kind: BackupConfiguration
metadata:
  name: sample-mgo-sh-backup2
  namespace: demo
spec:
  schedule: "*/5 * * * *"
  task:
    name: mg-backup-3.6
  repository:
    name: gcs-repo-custom
  target:
    ref:
      apiVersion: appcatalog.appscode.com/v1alpha1
      kind: AppBinding
      name: sample-mgo-sh-custom
  retentionPolicy:
    keepLast: 5
    prune: true
```

```console
$ kubectl create -f ./docs/examples/backup/sharding/standalone-backup.yaml
appbinding.appcatalog.appscode.com/sample-mgo-sh-custom created
repository.stash.appscode.com/gcs-repo-custom created
backupconfiguration.stash.appscode.com/sample-mgo-sh-backup2 created


$ kubectl get backupsession -n demo
NAME                              BACKUPCONFIGURATION    PHASE       AGE
sample-mgo-sh-backup-1563528902   sample-mgo-sh-backup   Succeeded   35s


$ kubectl get repository -n demo gcs-repo-custom
NAME              INTEGRITY   SIZE         SNAPSHOT-COUNT   LAST-SUCCESSFUL-BACKUP   AGE
gcs-repo-custom   true        22.160 KiB   4                1m                       2m
```

### Restore to a standalone database

No additional configuration is needed to restore the sharded cluster to a standalone database. Follow the normal procedure of restoring a MongoDB Database.

Standalone MongoDB,

```yaml
apiVersion: kubedb.com/v1alpha1
kind: MongoDB
metadata:
  name: restored-mongodb
  namespace: demo
spec:
  version: "3.6-v4"
  storageType: Durable
  databaseSecret:
    secretName: sample-mgo-sh-auth
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

RestoreSession crd object,

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
    name: mg-restore-3.6
  repository:
    name: gcs-repo-custom
  target:
    ref:
      apiVersion: appcatalog.appscode.com/v1alpha1
      kind: AppBinding
      name: restored-mongodb
  rules:
    - snapshots: [latest]
```

```console
$ kubectl create -f ./docs/examples/restore/sharding/restored-standalone.yaml
mongodb.kubedb.com/restored-mongodb created

$ kubectl get mg -n demo restored-mongodb
NAME               VERSION   STATUS         AGE
restored-mongodb   3.6-v4    Initializing   56s

$ kubectl create -f ./docs/examples/restore/sharding/restoresession-standalone.yaml
restoresession.stash.appscode.com/sample-mongodb-restore created

$ kubectl get mg -n demo restored-mongodb
NAME               VERSION   STATUS         AGE
restored-mongodb   3.6-v4    Running   56s
```

Now, exec into the database pod and list available tables,

```console
$ kubectl get secrets -n demo sample-mgo-sh-auth -o jsonpath='{.data.\username}' | base64 -d
root

$ kubectl get secrets -n demo sample-mgo-sh-auth -o jsonpath='{.data.\password}' | base64 -d
JJPcMxNKJev0SzgX

$ kubectl exec -it -n demo restored-mongodb-0 bash

mongodb@restored-mongodb-0:/$ mongo admin -u root -p JJPcMxNKJev0SzgX

> show dbs
admin   0.000GB
config  0.000GB
local   0.000GB
newdb   0.000GB

> show users
{
	"_id" : "admin.root",
	"userId" : UUID("98fa7511-2ae0-4466-bb2a-f9a7e17631ad"),
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
{ "_id" : ObjectId("5d3064bf144a1b8fda04cd4f"), "name" : "batman" }

> exit
bye
```

So, from the above output, we can see the database `newdb` that we had created in the original database `sample-mgo-sh` is restored in the restored database `restored-mongodb`.

## Cleanup

To cleanup the Kubernetes resources created by this tutorial, run:

```console
kubectl delete restoresession -n demo sample-mgo-sh-restore sample-mongodb-restore
kubectl delete backupconfiguration -n demo sample-mgo-sh-backup sample-mgo-sh-backup
kubectl delete mg -n demo restored-mgo-sh restored-mgo-sh restored-mongodb
kubectl delete repository -n demo gcs-repo-sharding gcs-repo-custom


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
