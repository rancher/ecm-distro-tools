# rpm

The rpm utility manages RPM packages in S3 buckets by creating and maintaining YUM repositories with proper metadata. It can sign RPMs, create repository metadata, merge with existing repositories, and handle complete repository rebuilds.

The tool requires a Linux environment with `createrepo_c`, `mergerepo_c`, and optionally `rpmsign`/`gpg` for package signing.

### Flags

| **Flag**                            | **Description**                                                                                                                                                                                      | **Required** |
| ----------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `bucket`, `b`                       | S3 bucket name where the RPM repository is stored                                                                                                                                                   | TRUE         |
| `prefix`, `p`                       | S3 prefix/path within the bucket (e.g., `dist/centos/8/x86_64`)                                                                                                                                    | FALSE        |
| `aws-access-key`                    | AWS Access Key ID for S3 authentication                                                                                                                                                             | TRUE         |
| `aws-secret-key`                    | AWS Secret Access Key for S3 authentication                                                                                                                                                         | TRUE         |
| `aws-region`                        | AWS Region where the S3 bucket is located (default: `us-east-1`)                                                                                                                                   | FALSE        |
| `visibility`                        | S3 ACL for uploaded objects: `private` or `public` (default: `private`)                                                                                                                            | FALSE        |
| `sign`                             | Sign RPMs and repository metadata with GPG                                                                                                                                                          | FALSE        |
| `sign-pass`                        | Passphrase for GPG signing (if empty, will prompt interactively)                                                                                                                                   | FALSE        |
| `rebuild`                          | Rebuild entire repository from scratch using existing S3 RPMs                                                                                                                                       | FALSE        |

### Examples

* Upload new RPMs to an empty S3 repository:
```sh
rpm --bucket my-rpm-bucket --prefix dist/centos/9/x86_64 \
    --aws-access-key yourAccessKey \
    --aws-secret-key yourSecretKey \
    package1.rpm package2.rpm package3.rpm
```

* Add RPMs to existing repository with signing:
```sh
rpm --bucket my-rpm-bucket --prefix dist/centos/9/x86_64 \
    --aws-access-key yourAccessKey \
    --aws-secret-key yourSecretKey \
    --sign --sign-pass "mypassphrase" \
    new-package.rpm
```

* Rebuild entire repository from S3 with signing:
```sh
rpm --bucket my-rpm-bucket --prefix dist/centos/9/x86_64 \
    --aws-access-key yourAccessKey \
    --aws-secret-key yourSecretKey \
    --aws-region us-west-2 \
    --rebuild --sign --sign-pass "mypassphrase"
```

* Upload with public visibility for a different AWS region:
```sh
rpm --bucket my-public-rpm-bucket --prefix releases/stable \
    --aws-access-key yourAccessKey \
    --aws-secret-key yourSecretKey \
    --aws-region eu-west-1 \
    --visibility public \
    --sign \
    production-ready.rpm
```

### How it works

1. **New Repository**: If no repository exists in S3, creates new metadata and uploads RPMs
2. **Existing Repository**: Downloads existing metadata, merges with new RPMs using `mergerepo_c`, and uploads updated repository
3. **Rebuild Mode**: Downloads all existing RPMs from S3, rebuilds complete metadata with `createrepo_c`
4. **Signing**: Optionally signs individual RPMs with `rpmsign` and repository metadata with `gpg`

### Prerequisites

- Linux environment with YUM repository tools:
  - `createrepo_c` (repository metadata creation)
  - `mergerepo_c` (repository merging)
  - `rpmsign` and `gpg` (for signing, optional)
- AWS credentials with S3 read/write permissions
- Valid GPG key configured for signing (if using `--sign`)

## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
