# Plugins

projx ships launcher plugins for Noctalia v5 and DMS (DankMaterialShell).
Both call `pjx list --format=delimited` and `pjx open`.

## Noctalia v5

```sh
noctalia msg plugins source add dev path /path/to/projx/plugins/noctalia
noctalia msg plugins enable projx/project-manager
```

Then open the Noctalia launcher and type `/pjx`.

There is also a bar widget (`plugins/noctalia/widget.luau`) showing the
current project name in the bar.

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