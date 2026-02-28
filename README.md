# Terraform State management using Git

Git as Terraform backend? Seriously? I know, might sound like a stupid idea at first, but let me try to convince you [why](#why-storing-state-in-git) it's not always the case.

## Table of Contents

- [Terraform State management using Git](#terraform-state-management-using-git)
  - [Table of Contents](#table-of-contents)
  - [Getting Started](#getting-started)
    - [Installation](#installation)
      - [Brew](#brew)
      - [From Release](#from-release)
      - [As Docker Image](#as-docker-image)
      - [As Github Action](#as-github-action)
      - [From Sources](#from-sources)
    - [Usage](#usage)
      - [Wrapper Mode](#wrapper-mode)
      - [Hashicorp Configuration Language (HCL) Mode](#hashicorp-configuration-language-hcl-mode)
      - [Standalone Terraform HTTP Backend Mode](#standalone-terraform-http-backend-mode)
    - [Wrappers CLI](#wrappers-cli)
    - [Configuration](#configuration)
    - [Git Credentials](#git-credentials)
    - [State Encryption](#state-encryption)
      - [`sops`](#sops)
        - [PGP](#pgp)
        - [AWS KMS](#aws-kms)
        - [GCP KMS](#gcp-kms)
        - [Hashicorp Vault](#hashicorp-vault)
        - [Age](#age)
      - [AES256](#aes256)
    - [Running backend remotely](#running-backend-remotely)
    - [TLS](#tls)
    - [Basic HTTP Authentication](#basic-http-authentication)
    - [Why not native Terraform Backend](#why-not-native-terraform-backend)
  - [Why storing state in Git](#why-storing-state-in-git)
  - [Proposed solution](#proposed-solution)
    - [Lock](#lock)
    - [Check existing Lock](#check-existing-lock)
    - [Unlock](#unlock)
    - [Get state](#get-state)
    - [Update state](#update-state)
    - [Delete state](#delete-state)

## Getting Started

### Installation

#### Brew

Installation with [Brew](https://github.com/plumber-cd/terraform-backend-git/issues/8) is coming later.

#### From Release

Download a binary from [Releases](https://github.com/plumber-cd/terraform-backend-git/releases). All binaries built with GitHub Actions and you can inspect [how](.github/workflows/release.yml).

Don't forget to add it to your `PATH`.

#### As Docker Image

See <https://github.com/plumber-cd/terraform-backend-git/pkgs/container/terraform-backend-git>.

```bash
docker pull ghcr.io/plumber-cd/terraform-backend-git:latest
```

#### As Github Action

See <https://github.com/marketplace/actions/setup-terraform-backend-git>.

```yaml
steps:
  - name: Setup terraform-backend-git
    uses: plumber-cd/setup-terraform-backend-git@v1
    with:
      version:
        0.1.2
  - name: Use terraform-backend-git
    run: terraform-backend-git version
```

#### From Sources

You can build it yourself, of course (and Go made it really easy):

```bash
go install github.com/plumber-cd/terraform-backend-git@${version}
```

Don't forget to add it to your `PATH`.

### Usage

The most easy to understand option is the `wrapper` mode.

#### Wrapper Mode

Assuming you've installed Terraform as well as this backend (and added it to your `PATH`), you can do this:

```bash
terraform-backend-git git \
  --repository https://github.com/my-org/tf-state \
  --ref master \
  --state my/state.json \
    terraform [any tf args] init|plan|apply [more tf args]
```

`terraform-backend-git` will act as a wrapper. It will start HTTP backend, generate Terraform configuration for it and save it to a `*.auto.tf` file. And then - it will just execute as-is everything you gave it to the right from `terraform` subcommand. After `terraform` exits - it will cleanup any `*.auto.tf` it created and shut down HTTP listener. You shouldn't be having any other backend configurations in your TF code, otherwise Terraform will fail with a conflict.

This mode is explained in more depth in the [wrapper CLI](#wrappers-cli) section.

#### Hashicorp Configuration Language (HCL) Mode

You could also create a `terraform-backend-git.hcl` config file and put it next to your `*.tf` code:

```hcl
git.repository = "https://github.com/my-org/tf-state"
git.ref = "main"
git.state = "my/state.json"
```

You can also specify custom path to the `hcl` config file using `--config` arg.

You can also have a mixed setup, where some parts of configuration comes from `terraform-backend-git.hcl` and some - from CLI arguments or even environment variables (see details below).

#### Standalone Terraform HTTP Backend Mode

Basically, you can run this backend as a standalone server (locally or remotely) as a daemon. You can either run it permanently, or have it started in your pipeline right before it is about to perform some Terraform actions.

```bash
terraform-backend-git &
```

Then, you just configure your Terraform code to use an [HTTP backend](https://www.terraform.io/docs/language/settings/backends/http.html).

Your Terraform backend configuration should be looking something like this:

```terraform
terraform {
  backend "http" {
    address = "http://localhost:6061/?type=git&repository=https://github.com/my-org/tf-state&ref=master&state=my/state.json"
    lock_address = "http://localhost:6061/?type=git&repository=https://github.com/my-org/tf-state&ref=master&state=my/state.json"
    unlock_address = "http://localhost:6061/?type=git&repository=https://github.com/my-org/tf-state&ref=master&state=my/state.json"
  }
}
```

Note that `lock_address` and `unlock_address` should both be explicitly defined. If they are not defined - Terraform assumes that the backend implementation does not support locking, so it will never attempt to lock the state, which might be dangerous and might lead to state file corruptions.

Now, just run Terraform and it will use the backend:

```bash
terraform init|plan|apply
```

When you're done, and if you want to stop the backend - it uses `pid` files to make it easier to stop:

```bash
terraform-backend-git stop
```

### Wrappers CLI

Command line syntax goes like this:

```bash
terraform-backend-git [backend options] <storage type> [storage options] <program> [any sub-process arguments]
```

For instance:

```bash
terraform-backend-git --access-logs git --state my/state.json terraform -detailed-exitcode -out=plan.out
#                                    |                            |
#                                    |                            \--- This is the program to run when HTTP backend is ready.
#                                    |                                 Everything to the right are as-is arguments to that program.
#                                    |
#                                    \-------------------- This is the name of the storage type to use.
#                                                          To the right are the arguments to control that storage settings.
#                                                          To the left are the arguments to control global backend settings.
```

Initially it is meant to only support `git` as a storage, hence the name of it included `git`. But later on it was realized that a pluggable architecture would allow to create alternative storage implementations re-using same protocol, encryption and so on. So tat's why it feels like a duplication of `git`, maybe in the future we will just rename the project to a `terraform-http-backend`.

`terraform` is also there because in the future we may extend support to other tools such as (but not limited to) `terragrunt` and `terratest`.

### Configuration

CLI | `terraform-backend-git.hcl` | Environment Variable | TF HTTP backend config | Description
--- | --- | --- | --- | ---
`--repository` | `git.repository` | `TF_BACKEND_GIT_GIT_REPOSITORY` |`repository` | Required; Which repository to use for storing TF state?
`--ref` | `git.ref` | `TF_BACKEND_GIT_GIT_REF` |`ref` | Optional; Which branch to use in that `repository`? Default: `master`.
`--state` | `git.state` | `TF_BACKEND_GIT_GIT_STATE` | `state` | Required; Path to the state file in that `repository`.
`--amend` | `git.amend` | `TF_BACKEND_GIT_GIT_AMEND` | `amend` | Optional; whether to use git amend + force push to update state file.
`--config` | - | - | - | Optional; Path to the `hcl` config file.
`--address` | `address` | `TF_BACKEND_GIT_ADDRESS` | - | Optional; Local binding address and port to listen for HTTP requests. Only change the port, **do not change the address to `0.0.0.0` before you read [Running backend remotely](#running-backend-remotely)**. Default: `127.0.0.1:6061`.
`--access-logs` | `accessLogs` | `TF_BACKEND_GIT_ACCESSLOGS` | - | Optional; Set to `true` to enable HTTP access logs on backend. Default: `false`.

### Git Credentials

Both HTTP and SSH protocols are supported. Sensitive values can be provided either directly via environment variables or via `*_FILE` variants.

Variable | Description
--- | ---
`GIT_USERNAME` | Specify username for Git, only required for HTTP protocol.
`GIT_PASSWORD`/`GITHUB_TOKEN` | Git password or token for HTTP protocol. In case of token you still have to specify `GIT_USERNAME`.
`GIT_PASSWORD_FILE`/`GITHUB_TOKEN_FILE` | Path to a file containing Git password or token for HTTP protocol (file content is trimmed).
`SSH_AUTH_SOCK` | `ssh-agent` socket.
`SSH_PRIVATE_KEY` | Path to SSH key for Git access.
`StrictHostKeyChecking` | Optional; If set to `no`, will not require strict host key checking. Somewhat more secure way of using Git in automation is to use `ssh -T -oStrictHostKeyChecking=accept-new git@github.com` before starting any automation.

When using `*_FILE` variables for HTTP auth, the file contents are cached in-memory and re-read when the file changes. This allows short-lived tokens to rotate without restarting the backend.

Backend will determine which protocol you are using based on the `repository` URL.

For SSH, it will see if `ssh-agent` is running by looking into `SSH_AUTH_SOCK` variable, and if not - it will need a private key. It will try to use `~/.ssh/id_rsa` unless you explicitly specify a different path via `SSH_PRIVATE_KEY`.

Unfortunately `go-git` will not mimic real Git client and will not automatically pickup credentials from the environment, so this custom credentials resolver chain has been implemented since I'm lazy to research the "right" original Git client approach. It is recommended to use Git Credentials Helpers (aka `ASKPASS`).

### State Encryption

To enable encryption set the env var `TF_BACKEND_HTTP_ENCRYPTION_PROVIDER` to one of the following values:

- `sops`
- `aes`

We are using [`sops`](https://github.com/mozilla/sops) as encryption abstraction. `sops` supports many different encryption backends, but unfortunately it does not provide one stop API for all of them, so on our side we should define configuration and create binding for each. At the moment, we have following bindings for `sops` backends:

- PGP
- AWS KMS
- GCP KMS
- Hashicorp Vault

Before we integrated with `sops` - we had a basic AES256 encryption via static passphrase. It is no longer recommended, although might be useful in some limited scenarios. Basic AES256 encryption is using one shared key, and it encrypts entire JSON state file that it can no longer be read as JSON. `sops` supports various encryption-as-service providers such as AWS KMS and Hashicorp Vault Transit - meaning encryption can be safely performed without revealing private key to the encryption clients. That means keys can be easily rotated, access can be easily revoked and generally it dramatically reduces chances of the key leaks.

#### `sops`

`sops` supports [Shamir's Secret Sharing](https://github.com/mozilla/sops#214key-groups). You can configure multiple backends at once - each will be used to encrypt a part of the key. You can set `TF_BACKEND_HTTP_SOPS_SHAMIR_THRESHOLD` if you want to use a specific threshold - by default, all keys used for encryption will be required for decryption.

##### PGP

Use `TF_BACKEND_HTTP_SOPS_PGP_FP` to provide a comma separated PGP key fingerprints. Keys must be added to a local `gpg` in order to encrypt. Private part of the key must be present in order for decrypt.

##### AWS KMS

Use `TF_BACKEND_HTTP_SOPS_AWS_KMS_ARNS` to provide a comma separated list of KMS ARNs. AWS SDK will use standard [credentials provider chain](https://docs.aws.amazon.com/sdk-for-go/api/aws/credentials/) in order to automatically discover local credentials in standard `AWS_*` environment variables or `~/.aws`. You can optionally use `TF_BACKEND_HTTP_SOPS_AWS_PROFILE` to point it to a specific shared profile. You can also provide additional KMS encryption context using `TF_BACKEND_HTTP_SOPS_AWS_KMS_CONTEXT` - it is a comma separated list of `key=value` pairs.

##### GCP KMS

Use `TF_BACKEND_HTTP_SOPS_GCP_KMS_KEYS` to provide a comma separated list of GCP KMS IDs. Read [Encrypting using GCP KMS](https://github.com/getsops/sops#encrypting-using-gcp-kms) for further details.

##### Hashicorp Vault

Use `TF_BACKEND_HTTP_SOPS_HC_VAULT_URIS` to point it to the Vault Transit keys. It is a comma separated list of URLs in a form of `${VAULT_ADDR}/v1/transit/keys/key`, where `transit` is a name of Vault Transit mount and `key` is the name of the key in that mount. Under the hood Vault SDK is using standard credentials resolver to automatically discover Vault credentials in the environment, meaning you can either use `vault login` or set `VAULT_TOKEN` environment variable.

##### Age

Use `TF_BACKEND_HTTP_SOPS_AGE_RECIPIENTS` to provide a comma separated list of age public keys. Ensure that corresponding private key is located in `keys.txt` in a `sops` subdirectory of your user configuration directory. Read [Encrypting using age](https://github.com/getsops/sops#encrypting-using-age) for further details.

#### AES256

To enable state encryption, you can use `TF_BACKEND_HTTP_ENCRYPTION_PASSPHRASE` environment variable to set a passphrase. Backend will encrypt and decrypt (using AES256, server-side) all state files transparently before storing them in Git. If it fails to decrypt the file obtained from Git, it will assume encryption was not previously enabled and return it as-is. Note this doesn't encrypt the traffic at REST, as Terraform doesn't support any sort of encryption for HTTP backend. Traffic between Terraform and this backend stays unencrypted at all times.

### Running backend remotely

This can be done, as previously mentioned, but it is not recommended. Although latest versions of this backend do support TLS in-transit encryption as well as at-rest encryption via `sops` - it still doesn't support authentication beyond very basic HTTP auth with a single shared password. Exposed backend will not give much flexibility in terms of the user access control, so it isn't really secure.

It is hard to tell at the moment where feature requests from users and my own use cases will take this project next, bur originally it was designed to be a local-only thing. Once backends in Terraform [can be pluggable gRPC components](https://github.com/hashicorp/terraform/issues/5877), this backend was planned to be converted to a normal gRPC plugin and HTTP support was planned to be removed. Basically, the idea was to use HTTP until gRCP for backend implementations were not available.

You may probably get creative and use something like Istio or maybe Keycloack to add external layer of encryption, authentication and authorization.

If you are absolutely sure you want to run this backend in remote standalone mode - you need to run it with `--address=:6061` argument so the backend will bind to `0.0.0.0` and become remotely accessible, otherwise - it will only listen on `127.0.0.1`.

### TLS

You can set `TF_BACKEND_GIT_HTTPS_CERT` and `TF_BACKEND_GIT_HTTPS_KEY` pointing to your cert and a key files. This will make HTTP backend to start in TLS mode. If you are using self-signed certificate - you can also set `TF_BACKEND_GIT_HTTPS_SKIP_VERIFICATION=true` in a wrapper mode and that will enable `skip_cert_verification` in the terraform config (or configure it yourself for standalone mode).

### Basic HTTP Authentication

You can use `TF_BACKEND_GIT_HTTP_USERNAME` and `TF_BACKEND_GIT_HTTP_PASSWORD` environment variables to add an extra layer of protection. In `wrapper` mode, same environment variables will be used to render `*.auto.tf` config for Terraform, but if you are using backend in standalone mode - you will have to tell these credentials to the Terraform explicitly:

```terraform
terraform {
  backend "http" {
    ...
    username = "user"
    password = "pswd"
  }
}
```

Note that if either username or password changes - Terraform will consider this as a backend configuration change and will want to ask you to migrate the state. Since backend will not be accepting old credentials anymore - it will fail to `init` (can't read the "old" state). Consider running `init -reconfigure` or deleting your local `.terraform/terraform.tfstate` file to fix this issue.

### Why not native Terraform Backend

Unfortunately, Terraform Backends is not pluggable like Providers are, see <https://github.com/hashicorp/terraform/issues/5877>.

Due to this, I couldn't make a proper native Terraform backend implementation for Git, it should have been implemented and added to <https://github.com/hashicorp/terraform> code base. There is an open ticket to do it <https://github.com/hashicorp/terraform/issues/24603>, but it is unclear when this would happen ([if it will at all](https://github.com/hashicorp/terraform/issues/24603#issuecomment-613533258)). That said I figured this HTTP backend implementation might be useful for the time being.

## Why storing state in Git

So you must be wondering why is that I think storing Terraform state in Git might be such a wonderful idea.

There is one particular chicken-egg problem that I ran into again, and again, and again. As I tend to manage ALL my infrastructure with code (and usually it's Terraform) - among the supported backend types none would exist before I create it. With code. Starting to feel the problem?

Backend types that use managed object storages (like `s3`) having the least amount of dependencies (i.e. they require no VPC), so before creating this backend - that's what I was usually using. But even then the chicken-egg issue is still there - you'd need a bucket itself, probably some replication config, encryption, IAM... And then there's also DynamoDB for locking. Usually I'd express that in TF code and just apply it locally for the first time (bootstrap). And then I will manually push that state to newly created bucket. What if I want to automate AWS account creation with Terraform too? To make it fully automated, which is totally doable, it would require some amount of custom glue... And that glue cannot be packaged as a Terraform module.

And then what if I want to go multi-cloud? Well, then I either store my GCP and Azure state in AWS, or I use 3 different state storages. Which would complicate my pipelines and make things less portable overall.

To throw even more shit on the fan - I also use Terraform to manage my Git repositories (with GitHub provider). It's an infrastructure too, after all. With proper structure and layers of abstractions - my Terraform code alone may easily go over 10 repositories for even smallest projects, and managing repositories should not be a burden. I want every single repository to be unified and configured in the same way, i.e. access, protected branches, merging policies etc.

And then - think about other people who doesn't even have infrastructure (or access). They might want to use Terraform for something completely irrelevant to the infrastructure, as there are hundreds of [providers](https://www.terraform.io/docs/providers/index.html) out there. What if they need to store TF state and just not ready to get into infra/pipelines management business?

Often when I start a new project, I myself - don't have any infrastructure for it yet. I don't even have an AWS account yet. I just want to create a few initial repositories to start working on it. And then my choice as to the state management is usually limited to a local state, and then I'd have to commit that state manually to git. It's fine when I'm alone, but as soon as multiple people involved - it gets complicated (things like manually "locking" the state via chat, fancy PR merging rules etc). And remember - we don't even have any infra yet, so forget about CD and pipelines for now.

Of course - there's Terraform Cloud, which is basically exists to address that exact problem (among many other). It provides state management as a service. A great product which I absolutely love, but honestly for a small projects, that doesn't need (yet?) any of that complex logic and fancy pipelines - sounds like an expensive overkill. I just remote state management with locking, that's all. Besides, what if that project is a PoC that is not even guaranteed to stay alive for a long time? What if the nature of the project is actually a Terraform proof of concept with a simple goal to sell developers on using it? If no one knows for sure yet if they even need Terraform - no one will buy commercial version of it for sure. I had to wear a hat of a Terraform proponent and a pioneer multiple times during my career, and all of this usually was a huge barrier and an obstacle for me to even establish initial conversations about Terraform. Terraform state migrations are a piece of cake so we can take care of that much later, when we actually need it.

One day I realized something really simple. If I'm pushing my Terraform state to git anyway (initially during bootstrap) - why not just fully embrace that concept and just do it right? Why not split the state from the code, create a separated isolated Git repository for it, and use it transparently to the Terraform user? Why not, basically, make Git a backend storage for a real Terraform backend?

Even if I don't have any infra yet - I surely do have some git server. I do have some repositories somewhere to share the code, right? It might be some public cloud service like GitHub/GitLab/Bitbucket/etc, or maybe it's a service within my Org that already existed on-prem.

## Proposed solution

Below is a proposal as to how a native Git backend implementation would look like in Terraform. HTTP backend implementation in this repository, basically, implements this proposal.

Consider a separate Git repository designated just for the Terraform state files. It is used as a backend, i.e. the fact it's a git repository is hidden from the user and considered an implementation detail. That means user scenarios doesn't really involve interacting with Git repository using Git clients.

Git server access configuration would define who have access to manage the state, i.e. users will still need their Git credentials. State files can also be encrypted in Git at rest.

The backend configuration might be looking something like this:

```terraform
terraform {
  backend "git" {
    repository = "https://github.com/my-org/tf-state?ref=main"
    file = "path/to/state.json"
  }
}
```

State locking would be based on branches, as creating a new branch is atomic operation.

To acquire a lock - it would mean to push a branch named `locks/${file}`. The branch would need to have a file `${file}.lock` added and committed to it with a standard Terraform locking metadata in it. If pushing the branch fails with error saying that fast forward push is not possible - that would mean something else already acquired the lock. To check if the state currently locked - would mean to check if the branch currently exists remotely. To read the information about the current lock - would mean to pull that branch and read the `${file}.lock`. To unlock - would mean to simply delete that remote branch.

This implementation proposal for the state locking might sound little weird, but keep in mind that the aim was to avoid complex Git scenarios that would involve merging and conflict solving. This proposal is trying to keep local Git working tree fast-forwardable at all times. As Git repository for state files is not really meant to be used by people directly at all, so it should be fine if we diverge a little from Git common best practices here.

To visualize and make it easier to understand, below is how the TF scenarios would translate into the command line:

### Lock

```bash
# Checkout current ref requested by user and cleanup any leftovers
git reset --hard
git checkout ${ref}
git branch -D locks/${file}
# Pull latest remote state
git pull origin ${ref}
# Start a new locking branch
git checkout -b locks/${file}
# Save lock metadata
echo ${lock} > ${file}.lock
git add ${file}.lock
git commit -m "Lock ${file}"
git push origin locks/${file}
# If push failed saying that fast forward is not possible - something else had it already locked
```

### Check existing Lock

```bash
# Checkout current ref requested by user and cleanup any leftovers
git reset --hard
git checkout ${ref}
git branch -D locks/${file}
# Fetch locks
git fetch origin refs/heads/locks/*:refs/remotes/origin/locks/*
# Checkout the lock branch, if it fails - it wasn't locked
git checkout locks/${file}
# Check if it was locked by me
cat ${file}.lock
```

### Unlock

```bash
# First - use routine from above to check that it is currently locked and the lock author is me.
# Then - it's a matter of deleting the lock branch remotely
git push origin --delete locks/${file}
```

### Get state

```bash
# Checkout current ref requested by user and cleanup any leftovers
git reset --hard
git checkout ${ref}
# Pull latest
git pull origin ${ref}
# Read state
cat ${file}
```

### Update state

```bash
# First - use routine from above to check that it is currently locked and the lock author is me.
# Then - checkout current ref requested by user and cleanup any leftovers
git reset --hard
git checkout ${ref}
# Pull latest
git pull origin ${ref}
# Save state
echo ${state} > ${file}
git add ${file}
git commit -m "Update ${file}"
git push origin ${ref}
```

### Delete state

```bash
# First - use routine from above to check that it is currently locked and the lock author is me.
# Then - checkout current ref requested by user and cleanup any leftovers
git reset --hard
git checkout ${ref}
# Pull latest
git pull origin ${ref}
# Delete state
git rm -f ${file}
git commit -m "Delete ${file}"
git push origin ${ref}
```
