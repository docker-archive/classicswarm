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
[here](http://releases.k8s.io/release-1.0/docs/troubleshooting.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

# Troubleshooting

Sometimes things go wrong.  This guide is aimed at making them right.  It has two sections:
   * [Troubleshooting your application](user-guide/application-troubleshooting.md) - Useful for users who are deploying code into Kubernetes and wondering why it is not working.
   * [Troubleshooting your cluster](admin/cluster-troubleshooting.md) - Useful for cluster administrators and people whose Kubernetes cluster is unhappy.

You should also check the [known issues](user-guide/known-issues.md) for the release you're using.

# Getting help

If your problem isn't answered by any of the guides above, there are variety of ways for you to get help from the Kubernetes team.

## Questions

If you aren't familiar with it, many of your questions may be answered by the [user guide](user-guide/README.md).

We also have a number of FAQ pages:
   * [User FAQ](https://github.com/kubernetes/kubernetes/wiki/User-FAQ)
   * [Debugging FAQ](https://github.com/kubernetes/kubernetes/wiki/Debugging-FAQ)
   * [Services FAQ](https://github.com/kubernetes/kubernetes/wiki/Services-FAQ)

You may also find the Stack Overflow topics relevant:
   * [Kubernetes](http://stackoverflow.com/questions/tagged/kubernetes)
   * [Google Container Engine - GKE](http://stackoverflow.com/questions/tagged/google-container-engine)

# Help! My question isn't covered!  I need help now!

## Stack Overflow

Someone else from the community may have already asked a similar question or may be able to help with your problem. The Kubernetes team will also monitor [posts tagged kubernetes](http://stackoverflow.com/questions/tagged/kubernetes). If there aren't any existing questions that help, please [ask a new one](http://stackoverflow.com/questions/ask?tags=kubernetes)!

## <a name="slack"></a>Slack

The Kubernetes team hangs out on Slack in the `#kubernetes-users` channel.  You can participate in the Kubernetes team [here](https://kubernetes.slack.com).  Slack requires registration, but the Kubernetes team is open invitation to anyone to register [here](http://slack.kubernetes.io).  Feel free to come and ask any and all questions.

## Mailing List

The Google Container Engine mailing list is [google-containers@googlegroups.com](https://groups.google.com/forum/#!forum/google-containers)

## Bugs and Feature requests

If you have what looks like a bug, or you would like to make a feature request, please use the [Github issue tracking system](https://github.com/kubernetes/kubernetes/issues).

Before you file an issue, please search existing issues to see if your issue is already covered.

If filing a bug, please include detailed information about how to reproduce the problem, such as:
* Kubernetes version: `kubectl version`
* Cloud provider, OS distro, network configuration, and Docker version
* Steps to reproduce the problem

<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/troubleshooting.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
