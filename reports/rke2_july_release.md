Contacts
--------

Primary: [Matt Trachier](/display/~mtrachier) 

Backup: [Brooks Newberry](/display/~bnewberry) / [Werner Garcia](/display/~wgarcia) 

QA: [Justin Janes](/display/~jjanes) 

  

[RKE2 Slack Release Thread](https://suse.slack.com/archives/C02DNASKFQB/p1657646877944789)

[K8S v1.24.3 Slack Release Thread](https://kubernetes.slack.com/archives/CJH2GBF7Y/p1657636376398049)  
[K8S v1.23.9 Slack Release Thread](https://kubernetes.slack.com/archives/CJH2GBF7Y/p1657636342587989)  
[K8S v1.22.12 Slack Release Thread](https://kubernetes.slack.com/archives/CJH2GBF7Y/p1657635913906239https://kubernetes.slack.com/archives/CJH2GBF7Y/p1657635913906239)

### 

*   1[Contacts](#RKE2JulyRelease-Contacts)
    *   1.1[](#RKE2JulyRelease-315pxtruenone)
*   2[About](#RKE2JulyRelease-About)
*   3[R1 Prep (July 13th)](#RKE2JulyRelease-R1Prep(July13th))
*   4[RC-1 (July 13th)](#RKE2JulyRelease-RC-1(July13th))
*   5[RC-2 (July 18th)](#RKE2JulyRelease-RC-2(July18th))
*   6[R2 Prep (July 18th)](#RKE2JulyRelease-R2Prep(July18th))
*   7[RC-3 (July 19th)](#RKE2JulyRelease-RC-3(July19th))
*   8[GA](#RKE2JulyRelease-GA)
*   9[Finalization](#RKE2JulyRelease-Finalization)
*   10[About](#RKE2JulyRelease-About.1)
    *   10.1[Who is this Document For?](#RKE2JulyRelease-WhoisthisDocumentFor?)
    *   10.2[What should this document contain?](#RKE2JulyRelease-Whatshouldthisdocumentcontain?)

About
-----

|     |     |     |     |     |     |     |     |
| --- | --- | --- | --- | --- | --- | --- | --- |
| **Previous Version** | **New Version** | **Kubernetes Version** | **K8s Release Date** | **Code Freeze** | **Planned Release Date** | **Actual Release Date** | **Stable RPM Release Date** |
| [v1.24.2+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.24.2+rke2r1) | [v1.24.3+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.24.3+rke2r1) | [v1.24.3](https://github.com/kubernetes/kubernetes/releases/tag/v1.24.3) | [July 13th](https://github.com/kubernetes/sig-release/issues/1963) | July 8th | July 21st | July 21st | July 22nd |
| [v1.23.8+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.23.8+rke2r1) | [v1.23.9+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.23.9+rke2r1) | [v1.23.9](https://github.com/kubernetes/kubernetes/releases/tag/v1.23.9) | [July 13th](https://github.com/kubernetes/sig-release/issues/1962) | July 8th | July 21st | July 21st | July 22nd |
| [v1.22.11+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.22.11+rke2r1) | [v1.22.12+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.22.12+rke2r1) | [v1.22.12](https://github.com/kubernetes/kubernetes/releases/tag/v1.22.12) | [July 13th](https://github.com/kubernetes/sig-release/issues/1961) | July 8th | July 21st | July 21st | July 22nd |

  
[RKE2 v1.24.3 Milestone](https://github.com/rancher/rke2/milestone/107)  
[RKE2 v1.23.9 Milestone](https://github.com/rancher/rke2/milestone/106)  
[RKE2 v1.22.12 Milestone](https://github.com/rancher/rke2/milestone/105)

R1 Prep (July 13th)
-------------------

| Hardened Kubernetes Releases | Kubernetes CI | Image Links | RKE2 PR | RKE2 PR CI | RKE2 Publish CI |
| --- | --- | --- | --- | --- | --- |
| Hardened Kubernetes Releases | Kubernetes CI | Image Links | RKE2 PR | RKE2 PR CI | RKE2 Publish CI |
| --- | --- | --- | --- | --- | --- |
| [v1.24.3-rke2r1-build20220713](https://github.com/rancher/image-build-kubernetes/releases/tag/v1.24.3-rke2r1-build20220713) | [v1.24](https://drone-publish.rancher.io/rancher/image-build-kubernetes/222) | [v1.24.3-rke2r1](https://hub.docker.com/r/rancher/hardened-kubernetes/tags?page=1&name=v1.24.3-rke2r1) | [v1.24](https://github.com/rancher/rke2/pull/3152) | [v1.24](https://drone-pr.rancher.io/rancher/rke2/3576) | [v1.24](https://drone-publish.rancher.io/rancher/rke2/1961) |
| [v1.23.9-rke2r1-build20220713](https://github.com/rancher/image-build-kubernetes/releases/tag/v1.23.9-rke2r1-build20220713) | [v1.23](https://drone-publish.rancher.io/rancher/image-build-kubernetes/223) | [v1.23.9-rke2r1](https://hub.docker.com/r/rancher/hardened-kubernetes/tags?page=1&name=v1.23.9-rke2r1) | [v1.23](https://github.com/rancher/rke2/pull/3151) | [v1.23](https://drone-pr.rancher.io/rancher/rke2/3575) | [v1.23](https://drone-publish.rancher.io/rancher/rke2/1960) |
| [v1.22.12-rke2r1-build20220713](https://github.com/rancher/image-build-kubernetes/releases/tag/v1.22.12-rke2r1-build20220713) | [v1.22](https://drone-publish.rancher.io/rancher/image-build-kubernetes/224) | [v1.22.12-rke2r1](https://hub.docker.com/r/rancher/hardened-kubernetes/tags?page=1&name=v1.22.12-rke2r1) | [v1.22](https://github.com/rancher/rke2/pull/3150) | [v1.22](https://drone-pr.rancher.io/rancher/rke2/3577) | [v1.22](https://drone-publish.rancher.io/rancher/rke2/1959) |

RC-1 (July 13th)
----------------

| Previous Merge CI | RKE2 Releases | RKE2 Releases CI | RKE2 System agent Installer CI | RKE2 Upgrade CI | KDM PR updated? | RKE2 Packages Releases | RKE2 Packages Releases CI |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Previous Merge CI | RKE2 Releases | RKE2 Releases CI | RKE2 System agent Installer CI | RKE2 Upgrade CI | KDM PR updated? | RKE2 Packages Releases | RKE2 Packages Releases CI |
| --- | --- | --- | --- | --- | --- | --- | --- |
| [998add17](https://drone-publish.rancher.io/rancher/rke2/1961) | [v1.24.3-rc1+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.24.3-rc1%2Brke2r1) | [v1.24.3](https://drone-publish.rancher.io/rancher/rke2/1964) | [v1.24](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/250) | [v1.24](https://drone-publish.rancher.io/rancher/rke2-upgrade/373) | yes | [v1.24.3-rc1+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.24.3-rc1%2Brke2r1.testing.0) | [v1.24](https://drone-publish.rancher.io/rancher/rke2-packaging/580) |
| [dc257287](https://drone-publish.rancher.io/rancher/rke2/1960) | [v1.23.9-rc1+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.23.9-rc1%2Brke2r1) | [v1.23.9](https://drone-publish.rancher.io/rancher/rke2/1963) | [v1.23](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/249) | [v1.23](https://drone-publish.rancher.io/rancher/rke2-upgrade/372) | yes | [v1.23.9-rc1+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.23.9-rc1%2Brke2r1.testing.0) | [v1.23](https://drone-publish.rancher.io/rancher/rke2-packaging/579) |
| [7946d8ad](https://drone-publish.rancher.io/rancher/rke2/1959) | [v1.22.12-rc1+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.22.12-rc1%2Brke2r1) | [v1.22.12](https://drone-publish.rancher.io/rancher/rke2/1962) | [v1.22](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/248) | [v1.22](https://drone-publish.rancher.io/rancher/rke2-upgrade/371) | yes | [v1.22.12-rc1+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.22.12-rc1+rke2r1.testing.0) | [v1.22](https://drone-publish.rancher.io/rancher/rke2-packaging/578) |

RC-2 (July 18th)
----------------

| Previous Merge CI | RKE2 Releases | RKE2 Releases CI | RKE2 System agent Installer CI | RKE2 Upgrade CI | KDM PR updated? | RKE2 Packages Releases | RKE2 Packages Releases CI |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Previous Merge CI | RKE2 Releases | RKE2 Releases CI | RKE2 System agent Installer CI | RKE2 Upgrade CI | KDM PR updated? | RKE2 Packages Releases | RKE2 Packages Releases CI |
| --- | --- | --- | --- | --- | --- | --- | --- |
| [ddb03170](https://drone-publish.rancher.io/rancher/rke2/1967) | [v1.24.3-rc2+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.24.3-rc2%2Brke2r1) | [v1.24](https://drone-publish.rancher.io/rancher/rke2/1971) | [24](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/253) | [24](https://drone-publish.rancher.io/rancher/rke2-packaging/583) | yes | [v1.24.3-rc2+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.24.3-rc2%2Brke2r1.testing.0) | [24](https://drone-publish.rancher.io/rancher/rke2-packaging/583) |
| [ba9ef87d](https://drone-publish.rancher.io/rancher/rke2/1966) | [v1.23.9-rc2+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.23.9-rc2%2Brke2r1) | [v1.23](https://drone-publish.rancher.io/rancher/rke2/1970) | [23](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/252) | [23](https://drone-publish.rancher.io/rancher/rke2-packaging/582) | yes | [v1.23.9-rc2+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.23.9-rc2+rke2r1.testing.0) | [23](https://drone-publish.rancher.io/rancher/rke2-packaging/582) |
| [2ea976bd](https://drone-publish.rancher.io/rancher/rke2/1968) | [v1.22.12-rc2+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.22.12-rc2%2Brke2r1) | [v1.22](https://drone-publish.rancher.io/rancher/rke2/1969) | [22](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/251) | [22](https://drone-publish.rancher.io/rancher/rke2-packaging/581) | yes | [v1.22.12-rc2+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.22.12-rc2+rke2r1.testing.0) | [22](https://drone-publish.rancher.io/rancher/rke2-packaging/581) |

R2 Prep (July 18th)
-------------------

| Hardened Kubernetes Releases | Kubernetes CI | Image Links | RKE2 PR | RKE2 PR CI | RKE2 Publish CI |
| --- | --- | --- | --- | --- | --- |
| Hardened Kubernetes Releases | Kubernetes CI | Image Links | RKE2 PR | RKE2 PR CI | RKE2 Publish CI |
| --- | --- | --- | --- | --- | --- |
| [v1.24.3-rke2r2-build20220718](https://github.com/rancher/image-build-kubernetes/releases/tag/v1.24.3-rke2r2-build20220718) | [v1.24](https://drone-publish.rancher.io/rancher/image-build-kubernetes/227) | [v1.24.3-rke2r2](https://hub.docker.com/r/rancher/hardened-kubernetes/tags?page=1&name=v1.24.3-rke2r2) | [24](https://github.com/rancher/rke2/pull/3164) | [24](https://drone-pr.rancher.io/rancher/rke2/3587) | \--- |
| [v1.23.9-rke2r2-build20220718](https://github.com/rancher/image-build-kubernetes/releases/tag/v1.23.9-rke2r2-build20220718) | [v1.23](https://drone-publish.rancher.io/rancher/image-build-kubernetes/226) | [v1.23.9-rke2r2](https://hub.docker.com/r/rancher/hardened-kubernetes/tags?page=1&name=v1.23.9-rke2r2) | [23](https://github.com/rancher/rke2/pull/3163) | [23](https://drone-pr.rancher.io/rancher/rke2/3586) | \--- |
| [v1.22.12-rke2r2-build20220718](https://github.com/rancher/image-build-kubernetes/releases/tag/v1.22.12-rke2r2-build20220718) | [v1.22](https://drone-publish.rancher.io/rancher/image-build-kubernetes/225) | [v1.22.12-rke2r2](https://hub.docker.com/r/rancher/hardened-kubernetes/tags?page=1&name=v1.22.12-rke2r2) | [22](https://github.com/rancher/rke2/pull/3162) | [22](https://drone-pr.rancher.io/rancher/rke2/3585) | \--- |

R2 was not necessary

RC-3 (July 19th)
----------------

| Previous Merge CI | RKE2 Releases | RKE2 Releases CI | RKE2 System agent Installer CI | RKE2 Upgrade CI | KDM PR updated? | RKE2 Packages Releases | RKE2 Packages Releases CI |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Previous Merge CI | RKE2 Releases | RKE2 Releases CI | RKE2 System agent Installer CI | RKE2 Upgrade CI | KDM PR updated? | RKE2 Packages Releases | RKE2 Packages Releases CI |
| --- | --- | --- | --- | --- | --- | --- | --- |
| [bd4f6718](https://drone-publish.rancher.io/rancher/rke2/1972) | [v1.24.3-rc3+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.24.3-rc3+rke2r1) | [v1.24](https://drone-publish.rancher.io/rancher/rke2/1977) | [24](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/256) | [24](https://drone-publish.rancher.io/rancher/rke2-upgrade/379) | yes | [v1.24.3-rc3+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.24.3-rc3+rke2r1.testing.0) | [24](https://drone-publish.rancher.io/rancher/rke2/1977) |
| [2d206eba](https://drone-publish.rancher.io/rancher/rke2/1974) | [v1.23.9-rc3+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.23.9-rc3+rke2r1) | [v1.23](https://drone-publish.rancher.io/rancher/rke2/1976) | [23](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/255) | [23](https://drone-publish.rancher.io/rancher/rke2-upgrade/378) | yes | [v1.23.9-rc3+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.23.9-rc3+rke2r1.testing.0) | [23](https://drone-publish.rancher.io/rancher/rke2/1976) |
| [1b6d8ed2](https://drone-publish.rancher.io/rancher/rke2/1973) | [v1.22.12-rc3+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.22.12-rc3+rke2r1) | [v1.22](https://drone-publish.rancher.io/rancher/rke2/1975) | [22](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/254) | [22](https://drone-publish.rancher.io/rancher/rke2-upgrade/377) | yes | [v1.22.12-rc3+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.22.12-rc3+rke2r1.testing.0) | [22](https://drone-publish.rancher.io/rancher/rke2-packaging/584) |

GA
--

| RKE2 Releases | RKE2 Releases CI | RKE2 System agent Installer CI | RKE2 Upgrade CI | KDM PR updated? | RKE2 Packages Releases | RKE2 Packages Releases CI |
| --- | --- | --- | --- | --- | --- | --- |
| RKE2 Releases | RKE2 Releases CI | RKE2 System agent Installer CI | RKE2 Upgrade CI | KDM PR updated? | RKE2 Packages Releases | RKE2 Packages Releases CI |
| --- | --- | --- | --- | --- | --- | --- |
| [v1.24.3+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.24.3+rke2r1) | [1980](https://drone-publish.rancher.io/rancher/rke2/1980) | [259](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/259) | [382](https://drone-publish.rancher.io/rancher/rke2-upgrade/382) | yes | [v1.24.3+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.24.3+rke2r1.testing.0) | [589](https://drone-publish.rancher.io/rancher/rke2-packaging/589) |
| [v1.23.9+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.23.9+rke2r1) | [1979](https://drone-publish.rancher.io/rancher/rke2/1979) | [258](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/258) | [381](https://drone-publish.rancher.io/rancher/rke2-upgrade/381) | yes | [v1.23.9+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.23.9+rke2r1.testing.0) | [588](https://drone-publish.rancher.io/rancher/rke2-packaging/588) |
| [v1.22.12+rke2r1](https://github.com/rancher/rke2/releases/tag/v1.22.12+rke2r1) | [1978](https://drone-publish.rancher.io/rancher/rke2/1978) | [257](https://drone-publish.rancher.io/rancher/system-agent-installer-rke2/257) | [380](https://drone-publish.rancher.io/rancher/rke2-upgrade/380) | yes | [v1.22.12+rke2r1.testing.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.22.12+rke2r1.testing.0) | [587](https://drone-publish.rancher.io/rancher/rke2-packaging/587) |

Finalization
------------

| Validation Reports | Latest RPM Release | Latest RPM CI | Stable RPM Release | Stable RPM CI |
| --- | --- | --- | --- | --- |
| Validation Reports | Latest RPM Release | Latest RPM CI | Stable RPM Release | Stable RPM CI |
| --- | --- | --- | --- | --- |
| [v1.24.3+rke2r1 Minor Version Validation](/pages/viewpage.action?pageId=1039663777) | [v1.24.3+rke2r1.latest.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.24.3+rke2r1.latest.0) | [592](https://drone-publish.rancher.io/rancher/rke2-packaging/592) | [v1.24.3+rke2r1.stable.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.24.3+rke2r1.stable.0) | [593](https://drone-publish.rancher.io/rancher/rke2-packaging/593) |
| [v1.23.9+rke2r1 Minor Version Validation](/pages/viewpage.action?pageId=1040712332) | [v1.23.9+rke2r1.latest.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.23.9+rke2r1.latest.0) | [591](https://drone-publish.rancher.io/rancher/rke2-packaging/591) | [v1.23.9+rke2r1.stable.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.23.9+rke2r1.stable.0) | [594](https://drone-publish.rancher.io/rancher/rke2-packaging/594) |
| [v1.22.12+rke2r1 Minor Version Validation](/pages/viewpage.action?pageId=1040712349) | [v1.22.12+rke2r1.latest.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.22.12+rke2r1.latest.0) | [590](https://drone-publish.rancher.io/rancher/rke2-packaging/590) | [v1.22.12+rke2r1.stable.0](https://github.com/rancher/rke2-packaging/releases/tag/v1.22.12+rke2r1.stable.0) | [595](https://drone-publish.rancher.io/rancher/rke2-packaging/595) |

  
[KDM PR](https://github.com/rancher/kontainer-driver-metadata/pull/934)  
[Release Notes PR](https://github.com/rancherlabs/release-notes/pull/259)  
[Channel Server PR](https://github.com/rancher/rke2/pull/3179)  
[Channel Server PR CI](https://drone-pr.rancher.io/rancher/rke2/3600)  
[Channel Server Publish CI](https://drone-publish.rancher.io/rancher/rke2/1981)  
[Channel Server Output](https://update.rke2.io/v1-release/channels)

About
-----

### Who is this Document For?

This document has two audiences, the release team and any audience following their progress.

#### Release Team

The release team uses this document as a scratch pad to place very specific information about the steps necessary to release RKE2.  
It is very important to note that this isn't a detailed list of the process to follow, this is a scratch pad for version numbers and specific release information.  
The [rke2 release document](https://github.com/rancher/rke2/blob/master/developer-docs/upgrading_kubernetes.md#update-rke2) is where specific information about the process and steps to complete it are, this is only data relating to a specific release.

#### Release Audience

Any number of people may be interested in the progress or specific data of a release, this document is a central place for specific release data to land.

### What should this document contain?

This document should only contain Data specific to a release, it isn't for release process information or general information which may be similar for all releases.  
Things which help someone historically follow the release process might be found here, but it isn't the source of truth for the steps followed.  
  
This document can be viewed as a central place for disparate specific data related to a release to be viewed, as such,  
the table of contents should use absolute links and headers should be nouns (not verbs), this conveys the thought that this is a report, not a workflow.