# Usage & command reference

## Core commands

```sh
pjx scan [roots...] [--max-depth N] [--dry-run]   # auto-discover projects
pjx add <path> [--name N] [--editor E] [--tag T] [--group G] [--favorite]
pjx list [--tag T] [--group G] [--favorites] [--sort recent|name|path] [--refresh] [--json|--format=text|json|delimited]
pjx open <project> [--editor E] [--profile P] [--new-window] [--add]
pjx open --set <name> [--new-window]              # open a workspace set
pjx remove <project>
pjx rename <project> <new-name>
pjx favorite <project>                            # toggle favorite
pjx tag <project> [+tag|-tag ...]
pjx group <project> <group-name>
pjx set <project> [--name N] [--editor E] [--profile P] [--group G] [--favorite[=bool]] [--tag T] [--untag T]
pjx refresh [project]                             # refresh cached git status
pjx doctor [--prune]                              # check/prune missing paths
pjx profiles [list [--json]]                      # list VS Code profiles
pjx profiles create <name>                    # create a new VS Code profile
pjx config [get|set <key> <value>]
pjx export [file]                                 # export projects to JSON
pjx import-file <file>                            # import projects from JSON
pjx ws create|list|delete|show ...
pjx tui                                           # interactive picker
pjx completion bash|zsh|fish
pjx version
```

## Editor / profile

```sh
# Switch editor / profile for a project (persistent)
pjx set my-app --editor code --profile "Work Profile"
pjx set my-app --editor zed          # keeps profile; use --profile "" to clear
pjx set my-app --editor code --profile ""   # clear profile

# One-off override (does not change stored project)
pjx open my-app --editor vscode --profile "Personal"

# Set default profile per editor
pjx config set editor_profiles '{"code":"Default VSCode Profile"}'
```

## VS Code profiles

```sh
# List known profiles (from VS Code sync data + profile directories)
pjx profiles
pjx profiles list --json

# Create a new profile (launches VS Code with that profile)
pjx profiles create "My New Profile"

# Use a profile on a project
pjx set my-app --editor code --profile "My New Profile"
```

VS Code creates the profile automatically on first use of `--profile <name>`.
`pjx set --profile` warns if the profile is not among known ones.

## Shell completion

`pjx completion <shell>` prints a completion script to stdout — it must be
**sourced** or installed for your shell (printing alone does not enable it).

### Bash

```sh
# current session only
source <(pjx completion bash)

# permanent (system-wide, requires bash-completion)
pjx completion bash | sudo tee /etc/bash_completion.d/pjx >/dev/null

# permanent (per-user)
mkdir -p ~/.local/share/bash-completion/completions
pjx completion bash > ~/.local/share/bash-completion/completions/pjx
```

Open a new terminal (or `source ~/.bashrc`) after a permanent install.

### Zsh

```sh
# current session only
source <(pjx completion zsh)

# permanent
mkdir -p ~/.zfunc
pjx completion zsh > ~/.zfunc/_pjx
```

Make sure `~/.zfunc` is in your `fpath` **before** `compinit` in `~/.zshrc`:

```zsh
fpath=(~/.zfunc $fpath)
autoload -Uz compinit && compinit
```

Open a new terminal. If completions do not appear, rebuild the cache:
`rm -f ~/.zcompdump* && exec zsh`.

### Fish

```sh
# current session only
pjx completion fish | source

# permanent
mkdir -p ~/.config/fish/completions
pjx completion fish > ~/.config/fish/completions/pjx.fish
```

Fish loads files in `~/.config/fish/completions/` automatically in new sessions.

## Workspace sets

```sh
pjx ws create myset ~/projects/a ~/projects/b
pjx ws list
pjx ws show myset
pjx open --set myset --new-window
pjx ws delete myset
```

## TUI keybindings

- `/` search · `j`/`k` or arrows navigate · `g`/`G` top/bottom
- `Enter` open · `Ctrl+O` open in new window · `Ctrl+R` refresh
- `f` toggle favorite · `d` remove · `r` rename · `?` help · `Esc`/`q` quit
- Command mode: `:` then `config`, `import`, `scan`, `favorite`, `set`, `ws`, `doctor`, `refresh`, `help`