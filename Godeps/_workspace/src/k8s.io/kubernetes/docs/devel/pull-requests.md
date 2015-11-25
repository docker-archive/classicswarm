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
[here](http://releases.k8s.io/release-1.0/docs/devel/pull-requests.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->
Pull Request Process
====================

An overview of how we will manage old or out-of-date pull requests.

Process
-------

We will close any pull requests older than two weeks.

Exceptions can be made for PRs that have active review comments, or that are awaiting other dependent PRs.  Closed pull requests are easy to recreate, and little work is lost by closing a pull request that subsequently needs to be reopened.

We want to limit the total number of PRs in flight to:
* Maintain a clean project
* Remove old PRs that would be difficult to rebase as the underlying code has changed over time
* Encourage code velocity

Life of a Pull Request
----------------------

Unless in the last few weeks of a milestone when we need to reduce churn and stabilize, we aim to be always accepting pull requests.

Either the [on call](https://github.com/kubernetes/kubernetes/wiki/Kubernetes-on-call-rotation) manually or the [submit queue](https://github.com/kubernetes/contrib/tree/master/submit-queue) automatically will manage merging PRs.

There are several requirements for the submit queue to work:
* Author must have signed CLA ("cla: yes" label added to PR)
* No changes can be made since last lgtm label was applied
* k8s-bot must have reported the GCE E2E build and test steps passed (Travis, Shippable and Jenkins build)

Additionally, for infrequent or new contributors, we require the on call to apply the "ok-to-merge" label manually.  This is gated by the [whitelist](https://github.com/kubernetes/contrib/tree/master/submit-queue/whitelist.txt).

Automation
----------

We use a variety of automation to manage pull requests.  This automation is described in detail
[elsewhere.](automation.md)


<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/devel/pull-requests.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
