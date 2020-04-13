# Terraform State management using Git repository

- [Terraform State management using Git repository](#terraform-state-management-using-git-repository)
  - [Terraform HTTP backend to manage state in Git](#terraform-http-backend-to-manage-state-in-git)
    - [Usage](#usage)
    - [Why not native Terraform Backend](#why-not-native-terraform-backend)
  - [Why](#why)
  - [Proposed solution](#proposed-solution)
    - [Lock](#lock)
    - [CheckLock](#checklock)
    - [UnLock](#unlock)
    - [GetState](#getstate)
    - [UpdateState](#updatestate)
    - [DeleteState](#deletestate)

Git as Terraform backend? Seriously? I know, might sound like a stupid idea at first, but let me try to convince you [why](#why) it's actually not.

## Terraform HTTP backend to manage state in Git

### Usage

First, download a backend from [release](https://github.com/plumber-cd/terraform-backend-git/releases) page.

Alternatively, you can build it yourself:

```bash
go build github.com/plumber-cd/terraform-backend-git
```

Of course, you must be having Terraform installed already.

You should be good to Go:

```bash
terraform-backend-git git --repository git@github.com:my-org/tf-state.git --ref master --state my/state.json terraform init|plan|apply
```

`terraform-backend-git` will act as a wrapper - it will start a backend, generate `*.auto.tf` file in current working directory and fall back to terraform accordingly to your input. After done it will cleanup `*.auto.tf` file it created. The file would contain an HTTP backend configuration pointing to that backend instance, so you shouldn't be having any other backend configurations in your TF code.

Alternatively, you could have more control over the process if you are using something like `terragrunt`. Bottom line, your Terraform backend config should be looking like this:

```terraform
terraform {
  backend "http" {
    address = "http://localhost:6061/?type=git&repository=git@github.com:my-org/tf-state.git&ref=master&state=my/state.json"
    lock_address = "http://localhost:6061/?type=git&repository=git@github.com:my-org/tf-state.git&ref=master&state=my/state.json"
    unlock_address = "http://localhost:6061/?type=git&repository=git@github.com:my-org/tf-state.git&ref=master&state=my/state.json"
  }
}
```

Then you could use it like this:

```bash
terraform-backend-git &
terraform init|plan|apply
terraform-backend-git stop
```

Note that `lock_address` and `unlock_address` should be explicitly defined (both of them), otherwise Terraform would not make any locking or unlocking calls and assume that backend does not support locking and unlocking (how would locking be supported without unlocking?...).

`--ref` and `ref` is optional, by default it will be set to `master`.

`--state`/`state` is a path to the state file in the repository.

Both HTTP and SSH protocols are supported for Git. For HTTP credentials, please define `GIT_USERNAME` and either `GIT_PASSWORD` or `GIT_TOKEN` environment variables. For SSH, it will try to use `SSH_AUTH_SOCK` environment variable if defined (assuming `ssh-agent` has been started), otherwise it will need a private key file. You can define a path to it via `SSH_PRIVATE_KEY` environment variable, and if not defined it will try to use `~/.ssh/id_rsa`. Unfortunately `go-git` will not mimic Git client and will not automatically pickup credentials from the environment, so this custom credentials resolver chain has been implemented since I'm lazy to research the "right" original Git client approach.

To enable backend encryption, you can use `TF_BACKEND_HTTP_ENCRYPTION_PASSPHRASE` environment variable to set a passphrase. Backend will encrypt and decrypt (using AES256) all state files transparently before storing them in Git. If it fails to decrypt the file obtained from Git, it will assume encryption was not previously enabled and return it as-is. Note this doesn't encrypt the traffic at REST, as Terraform doesn't support any sort of encryption. Traffic between Terraform and this backend stays unencrypted at all times.

This backend could be started standalone and remotely, but I would not recommend doing that.

Besides that Terraform does not perform any encryption before sending the state to HTTP backend, there is also no authentication whatsoever. Running remotely accessible backend like this would not be secure - anyone who can make HTTP calls to it would be able to get, update or delete your state files with no credentials. Make sure you do not open the port in your firewall for remote connections. By default it would start on port `6061` and would use `127.0.0.1` as the binding address, so that nothing would be able to connect remotely. That would still not protect you from local loop interface traffic interceptions, but that's the best we can do for now, either until this implementation gets into Terraform as a native Backend implementation, or Backends become a pluggable options, or gRCP backend being implemented or Terraform adds some auth/encryption options to the HTTP backend protocol, or some other miracle.

You may get creative and wrap backend traffic into API Gateway or ServiceMesh like Istio, to add encryption and authentication, then you will want to use this option `--address=:6061` so the backend will bind to `0.0.0.0` and become remotely accessible. You can change the port that way too, i.e. `--address=127.0.0.1:6062`.

Use `--access-logs` to enable HTTP access logs.

### Why not native Terraform Backend

Unfortunately, Terraform Backends is not pluggable like Providers are, see https://github.com/hashicorp/terraform/issues/5877.

Due to this, I couldn't make a proper native Terraform backend implementation for Git on a side, it should be implemented and added to https://github.com/hashicorp/terraform code base. There is an open ticket to do it https://github.com/hashicorp/terraform/issues/24603, but it is unclear when this would happen (if it will at all). That said I figured this HTTP backend implementation might be useful for now.

## Why

So you must be wondering why storing Terraform state in Git might be such a good idea.

I often face the same chicken-egg issue, again and again and again... as I tend to manage ALL my infrastructure with Code (and usually it's Terraform), among the supported backend types none would exist before I create it. With code. Feel the problem?

Backend types that uses managed object storages (like `s3`) having the least amount of dependencies (i.e. no VPC and etc), so I usually was leaning towards using them, but even then the chicken-egg issue is still there. Usually I'm having some generic TF modules for my `s3` and `dynamodb` implementations, that I use then as dependencies to my top-level root module that ultimately defines and manages my TF state backend. And I would usually apply it for a first time (bootstrapping) using a local state file, and then manually push that state to newly created backend. To make it fully automated, which is totally doable, it would require some amount of custom glue... and would cause complications for destroy/recreate type of operations. Applying (specifically, bootstrapping) this specific piece of infrastructure would require some custom logic specific to only that piece of infrastructure, and that logic cannot be packaged as a Terraform module. So, TL;DR: the problem is kinda still there, I just kinda learned how to live with it. Sounds familiar? Keep reading.

To throw more shit on the fan, I also use Terraform to manage my Git repositories (with GitHub or Bitbucket provider). It's an infrastructure too, after all. With proper structure and abstractions Terraform code alone may easily be over 50 repositories for even smallest projects, and managing repositories should not be a burden. I want every single repository to be unified and configured same way, i.e. access/protected branches/merging policies/etc. And often when I start a project, I don't have any infrastructure for it yet, I don't even have an AWS account or whatever yet, I just want to create a few initial repositories to start working on it. And then my choice as to the state management usually limited to a local state and committing that state to git. It's fine when I'm alone, but as soon as multiple people involved it gets complicated (things like manually "locking" the state via chat, fancy PR merging rules, and etc). And remember we don't even have any infra yet, so forget about CD and pipelines for now.

Of course there's Terraform Cloud/Enterprise addressing specifically that issue. A great product which I absolutely love, but honestly for a small projects, that doesn't need (yet?) any of that complex logic and fancy pipelines, just remote state management with locking - sounds like an expensive overhead. Besides at the beginning of a new project, maybe even a PoC that doesn't even guaranteed to stay for a long time, maybe even a PoC to prove Terraform is a right tool so no one really yet sold on the idea to buy anything for it, do you really think the very first and right thing to do should be to go through procurement and legal processes to get a contract signed with a 3rd party? Sounds like an obstacle and a yak shaving to me. Migrating Terraform state is a piece of cake so we can take care of that much later, when we need it.

One day I came to a simple conclusion. If I'm committing my Terraform state to git anyway (at least initially) - why not just fully embrace that concept and do it right? Split the state from the code, dedicate separate isolated Git repository just for the state, and use it transparently to the user - basically make Git a real Terraform backend. That would actually solve my chicken-egg problem.

Or, would it? Well, maybe not entirely, more like shift it elsewhere. Even if I don't have any infra yet - I surely do have some git server. If I'm about to produce some Terraform modules, I'm surely have some Git location to store them, reference them as dependencies from one another, etc... I'm surely have some space for my team to collaborate on these modules. It might be some public cloud service like GitHub/GitLab/Bitbucket/CodeCommit/etc, or maybe it's a service within my Org that already existed elsewhere, like on-prem or whatever. Sure, technically, the chicken-egg problem isn't going completely away, sounds like a git server needs to be there for you somehow before you start, but c'mon what are the chances you don't have Git server at the start of a new infra project and you would need to setup it just for the sake of TF? Sounds like the chances are that problem would have been solved somehow way before you get to Terraform, so I would consider this approach a proper chicken-egg resolution for Terraform state management.

I'm not trying to make it look like this is the right and correct way for storing state files, it's probably not. But for the initial stages of the project for the sake of the solving that chicken-egg problem - it would do.

And then think about other engineers who doesn't have infrastructure or access to it, like application developers. They might want to use Terraform for something completely irrelevant to the infrastructure, there's hundreds of [providers](https://www.terraform.io/docs/providers/index.html) out there, what if they need to store a state and not ready to get into infra/pipelines management business? On the other side, everybody has access to git. Well, most of us likely do. So...

## Proposed solution

Below is a proposal as to how a native Git backend implementation would look like in Terraform. This HTTP backend implements this proposal, so it would be easier to transfer the code at some point.

Consider a separate Git repository designated just for the Terraform state files. It is used as a backend, i.e. the fact it's a git repository is hidden from the user and considered an implementation detail. That means user scenarios doesn't involve interacting with this repository using Git clients. Git server access configuration would define who have access to manage the state, i.e. users will still need their Git credentials. If Git server access control capabilities isn't enough to meet security requirements, state files might be encrypted on backend, there would be no reason for them to be stored in open text in Git. Storing a state file would be as simple as committing and pushing it to the repository.

Theoretically the same repository with code can be also used as state management. But you are likely will want to use some branch protection and/or PRs, so this might work for your specific use case but is not recommended.

The backend configuration might be looking something like this:

```terraform
terraform {
  backend "git" {
    repository = "git@github.com:my-org/tf-state.git?ref=master"
    file = "path/to/state.json"
  }
}
```

State locking would be based on branches. The following implementation proposal for the state locking might sound little weird, but keep in mind as you read it that the aim was to avoid complex Git scenarios that would involve merging and conflict solving, like it wasn't complex enough to use Git as a Terraform state management backend to begin with. This proposal trying to keep local Git working tree fast-forwardable at all times. Git repository in subject is not meant to be used by people directly after all, so it's fine if we do not follow some Git common sense here.

To acquire a lock would mean to push a branch named `locks/${file}`. The branch would need to have a file `${file}.lock` added and committed to it with the standard Terraform locking metadata. If pushing the branch fails with error saying that fast forward push is not possible, that would mean somebody else already acquired the lock. That would make a locking operation truly atomic. To check if the state currently locked is to see if the branch currently exists remotely. To read the information about the current lock, would mean to pull that branch and read the `${file}.lock`. To unlock would mean to delete that remote branch.

To visualize and make it easier to understand, below is how the TF scenarios would translate into the command lines:

### Lock

```bash
# Checkout current ref requested by user and cleanup any leftowers
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
# If push failed saying fast forward not possible - somebody else had it already locked
```

### CheckLock

```bash
# Checkout current ref requested by user and cleanup any leftowers
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

### UnLock

```bash
CheckLock
# Now it's a matter of deleting the lock branch remotely
git push origin --delete locks/${file}
```

### GetState

```bash
# Checkout current ref requested by user and cleanup any leftowers
git reset --hard
git checkout ${ref}
# Pull latest
git pull origin ${ref}
# Read state
cat ${file}
```

### UpdateState

```bash
CheckLock
# Checkout current ref requested by user and cleanup any leftowers
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

### DeleteState

```bash
CheckLock
# Checkout current ref requested by user and cleanup any leftowers
git reset --hard
git checkout ${ref}
# Pull latest
git pull origin ${ref}
# Delete state
git rm -f ${file}
git commit -m "Delete ${file}"
git push origin ${ref}
```
