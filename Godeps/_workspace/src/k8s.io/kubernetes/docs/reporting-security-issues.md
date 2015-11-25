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
[here](http://releases.k8s.io/release-1.0/docs/reporting-security-issues.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

# Security

If you believe you have discovered a vulnerability or a have a security incident to report, please follow the steps below. This applies to Kubernetes releases v1.0 or later.

To watch for security and major API announcements, please join our [kubernetes-announce](https://groups.google.com/forum/#!forum/kubernetes-announce) group.

## Reporting a security issue

To report an issue, please:
- Submit a bug report [here](http://goo.gl/vulnz).
  - Select “I want to report a technical security bug in a Google product (SQLi, XSS, etc.).”
  - Select “Other” as the Application Type.
- Under reproduction steps, please additionally include
  - the words "Kubernetes Security issue"
  - Description of the issue
  - Kubernetes release (e.g. output of `kubectl version` command, which includes server version.)
  - Environment setup (e.g.  which "Getting Started Guide" you followed, if any; what node operating system used; what service or software creates your virtual machines, if any)

An online submission will have the fastest response; however, if you prefer email, please send mail to security@google.com. If you feel the need, please use the [PGP public key](https://services.google.com/corporate/publickey.txt) to encrypt communications.


<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/reporting-security-issues.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
