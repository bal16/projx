# Plugins

projx ships launcher plugins for Noctalia v5 and DMS (DankMaterialShell).
Both call `pjx list --format=delimited` and `pjx open`.

## Noctalia v5

The Noctalia plugin lives in `plugins/project-manager/` (the directory name
must match the plugin id suffix `projx/project-manager`; Noctalia resolves
path sources by that name).

```sh
# 1. pjx must be on PATH (the plugin manifest declares it as a dependency)
go install github.com/bal16/projx/cmd/pjx@latest   # or: make build && cp bin/pjx ~/.local/bin/

# 2. Register the plugin's parent directory as a path source
noctalia msg plugins source add dev path /path/to/projx/plugins

# 3. Enable it
noctalia msg plugins enable projx/project-manager
```

Then open the Noctalia launcher and type `/pjx`.

There is also a bar widget (`plugins/project-manager/widget.luau`) showing the
current project name in the bar.

Settings (gear icon on the plugin row in Noctalia settings):

- `new_window` — open projects in a new editor window instead of reusing the
  existing one.

Check the plugin with the offline linter:

```sh
noctalia plugins lint /path/to/projx/plugins/project-manager
```

## DMS (DankMaterialShell)

```sh
cp -r /path/to/projx/plugins/dms ~/.config/dms/plugins/projxLauncher
dms restart
```

Then open the DMS launcher and type `pjx`.

There is also a bar widget (`plugins/dms-widget/`):

```sh
cp -r /path/to/projx/plugins/dms-widget ~/.config/dms/plugins/projxCurrentProject
dms restart
```

## Plugin contract (DMS)

DMS launcher expects `getItems(query)` returning items with
`{name, comment, action, icon, categories}` and `executeItem(item)`.
See `plugins/dms/ProjxLauncher.qml` for the reference implementation.