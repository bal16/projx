# Configuration & data locations

## Data locations (production defaults)

- Database: `~/.local/share/projx/projects.db` (SQLite, WAL mode)
- Config: `~/.config/projx/config.json`
- Cache: `~/.cache/projx/` (reserved for future remote metadata cache)

## Isolating dev vs production

Production uses the default XDG paths above. For development/CI, isolate with
env vars so test data never pollutes your real DB/config:

```sh
# All-in-one isolation (recommended for dev)
export PROJX_HOME=/tmp/projx-dev
pjx scan ~/scratchpad

# Or granular overrides
export PROJX_DATA_DIR=/tmp/projx-data
export PROJX_CONFIG_DIR=/tmp/projx-config
export PROJX_CACHE_DIR=/tmp/projx-cache
```

Resolution order:
- Data: `PROJX_DATA_DIR` > `PROJX_HOME/data` > `XDG_DATA_HOME/projx` > `~/.local/share/projx`
- Config: `PROJX_CONFIG_DIR` > `PROJX_HOME/config` > `XDG_CONFIG_HOME/projx` > `~/.config/projx`
- Cache: `PROJX_CACHE_DIR` > `PROJX_HOME/cache` > `XDG_CACHE_HOME/projx` > `~/.cache/projx`

## Config keys

```sh
pjx config                          # show full config
pjx config get default_editor
pjx config set default_editor zed
pjx config set scan_roots '["~/projects","~/code"]'
pjx config set scan_max_depth 3
pjx config set editor_profiles '{"code":"Default VSCode Profile"}'
```

See `docs/examples/config.json` for a full example.