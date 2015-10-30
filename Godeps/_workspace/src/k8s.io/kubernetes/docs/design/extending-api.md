<!-- BEGIN MUNGE: UNVERSIONED_WARNING -->

<!-- BEGIN STRIP_FOR_RELEASE -->

<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">

<h2>PLEASE NOTE: This document applies to the HEAD of the source tree</h2>

If you are using a released version of Kubernetes, you should
refer to the docs that go with that version.

<strong>
The latest 1.0.x release of this document can be found
[here](http://releases.k8s.io/release-1.0/docs/design/extending-api.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

# Adding custom resources to the Kubernetes API server

This document describes the design for implementing the storage of custom API types in the Kubernetes API Server.


## Resource Model

### The ThirdPartyResource

The `ThirdPartyResource` resource describes the multiple versions of a custom resource that the user wants to add
to the Kubernetes API.  `ThirdPartyResource` is a non-namespaced resource, attempting to place it in a resource
will return an error.

Each `ThirdPartyResource` resource has the following:
   * Standard Kubernetes object metadata.
   * ResourceKind - The kind of the resources described by this third party resource.
   * Description - A free text description of the resource.
   * APIGroup - An API group that this resource should be placed into.
   * Versions - One or more `Version` objects.

### The `Version` Object

The `Version` object describes a single concrete version of a custom resource.  The `Version` object currently
only specifies:
   * The `Name` of the version.
   * The `APIGroup` this version should belong to.

## Expectations about third party objects

Every object that is added to a third-party Kubernetes object store is expected to contain Kubernetes
compatible [object metadata](../devel/api-conventions.md#metadata).  This requirement enables the
Kubernetes API server to provide the following features:
   * Filtering lists of objects via LabelQueries
   * `resourceVersion`-based optimistic concurrency via compare-and-swap
   * Versioned storage
   * Event recording
   * Integration with basic `kubectl` command line tooling.
   * Watch for resource changes.

The `Kind` for an instance of a third-party object (e.g. CronTab) below is expected to be
programmatically convertible to the name of the resource using
the following conversion.  Kinds are expected to be of the form `<CamelCaseKind>`, the
`APIVersion` for the object is expected to be `<domain-name>/<api-group>/<api-version>`.

For example `example.com/stable/v1`

`domain-name` is expected to be a fully qualified domain name.

'CamelCaseKind' is the specific type name.

To convert this into the `metadata.name` for the `ThirdPartyResource` resource instance,
the `<domain-name>` is copied verbatim, the `CamelCaseKind` is
then converted
using '-' instead of capitalization ('camel-case'), with the first character being assumed to be
capitalized.  In pseudo code:

```go
var result string
for ix := range kindName {
  if isCapital(kindName[ix]) {
    result = append(result, '-')
  }
  result = append(result, toLowerCase(kindName[ix])
}
```

As a concrete example, the resource named `camel-case-kind.example.com` defines resources of Kind `CamelCaseKind`, in
the APIGroup with the prefix `example.com/...`.

The reason for this is to enable rapid lookup of a `ThirdPartyResource` object given the kind information.
This is also the reason why `ThirdPartyResource` is not namespaced.

## Usage

When a user creates a new `ThirdPartyResource`, the Kubernetes API Server reacts by creating a new, namespaced
RESTful resource path.  For now, non-namespaced objects are not supported. As with existing built-in objects
deleting a namespace, deletes all third party resources in that namespace.

For example, if a user creates:

```yaml
metadata:
  name: cron-tab.example.com
apiVersion: experimental/v1alpha1
kind: ThirdPartyResource
description: "A specification of a Pod to run on a cron style schedule"
versions:
  - name: stable/v1
  - name: experimental/v2
```

Then the API server will program in two new RESTful resource paths:
   * `/thirdparty/example.com/stable/v1/namespaces/<namespace>/crontabs/...`
   * `/thirdparty/example.com/experimental/v2/namespaces/<namespace>/crontabs/...`


Now that this schema has been created, a user can `POST`:

```json
{
   "metadata": {
     "name": "my-new-cron-object"
   },
   "apiVersion": "example.com/stable/v1",
   "kind": "CronTab",
   "cronSpec": "* * * * /5",
   "image":     "my-awesome-chron-image"
}
```

to: `/third-party/example.com/stable/v1/namespaces/default/crontabs/my-new-cron-object`

and the corresponding data will be stored into etcd by the APIServer, so that when the user issues:

```
GET /third-party/example.com/stable/v1/namespaces/default/crontabs/my-new-cron-object`
```

And when they do that, they will get back the same data, but with additional Kubernetes metadata
(e.g. `resourceVersion`, `createdTimestamp`) filled in.

Likewise, to list all resources, a user can issue:

```
GET /third-party/example.com/stable/v1/namespaces/default/crontabs
```

and get back:

```json
{
   "apiVersion": "example.com/stable/v1",
   "kind": "CronTabList",
   "items": [
     {
       "metadata": {
         "name": "my-new-cron-object"
       },
       "apiVersion": "example.com/stable/v1",
       "kind": "CronTab",
       "cronSpec": "* * * * /5",
       "image":     "my-awesome-chron-image"
    }
   ]
}
```

Because all objects are expected to contain standard Kubernetes metadata fields, these
list operations can also use `Label` queries to filter requests down to specific subsets.

Likewise, clients can use watch endpoints to watch for changes to stored objects.


## Storage

In order to store custom user data in a versioned fashion inside of etcd, we need to also introduce a
`Codec`-compatible object for persistent storage in etcd.  This object is `ThirdPartyResourceData` and it contains:
   * Standard API Metadata
   * `Data`: The raw JSON data for this custom object.

### Storage key specification

Each custom object stored by the API server needs a custom key in storage, this is described below:

#### Definitions

   * `resource-namespace` : the namespace of the particular resource that is being stored
   * `resource-name`: the name of the particular resource being stored
   * `third-party-resource-namespace`: the namespace of the `ThirdPartyResource` resource that represents the type for the specific instance being stored.
   * `third-party-resource-name`: the name of the `ThirdPartyResource` resource that represents the type for the specific instance being stored.

#### Key

Given the definitions above, the key for a specific third-party object is:

```
${standard-k8s-prefix}/third-party-resources/${third-party-resource-namespace}/${third-party-resource-name}/${resource-namespace}/${resource-name}
```

Thus, listing a third-party resource can be achieved by listing the directory:

```
${standard-k8s-prefix}/third-party-resources/${third-party-resource-namespace}/${third-party-resource-name}/${resource-namespace}/
```


<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/design/extending-api.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
