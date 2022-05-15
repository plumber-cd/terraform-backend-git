## terraform-backend-git completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	terraform-backend-git completion fish | source

To load completions for every new session, execute once:

	terraform-backend-git completion fish > ~/.config/fish/completions/terraform-backend-git.fish

You will need to start a new shell for this setup to take effect.


```
terraform-backend-git completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -l, --access-logs      Log HTTP requests to the console
  -a, --address string   Specify the listen address (default "127.0.0.1:6061")
  -c, --config string    config file (default is terraform-backend-git.hcl)
```

### SEE ALSO

* [terraform-backend-git completion](terraform-backend-git_completion.md)	 - Generate the autocompletion script for the specified shell

###### Auto generated by spf13/cobra on 15-May-2022