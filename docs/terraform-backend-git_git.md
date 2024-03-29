## terraform-backend-git git

Start backend in Git storage mode and execute the wrapper

### Synopsis

It will also generate git_http_backend.auto.tf in current working directory pointing to this backend

### Options

```
  -d, --dir string          Change current working directory
  -h, --help                help for git
  -b, --ref string          Ref (branch) to use (default "master")
  -r, --repository string   Repository to use as storage
  -s, --state string        Ref (branch) to use
```

### Options inherited from parent commands

```
  -l, --access-logs      Log HTTP requests to the console
  -a, --address string   Specify the listen address (default "127.0.0.1:6061")
  -c, --config string    config file (default is terraform-backend-git.hcl)
```

### SEE ALSO

* [terraform-backend-git](terraform-backend-git.md)	 - Terraform HTTP backend implementation that uses Git as storage
* [terraform-backend-git git terraform](terraform-backend-git_git_terraform.md)	 - Run terraform while storage is running

###### Auto generated by spf13/cobra on 15-May-2022
