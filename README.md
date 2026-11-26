# devgc

Finds build outputs (and, if you ask, dependencies) of dev projects in a folder,
shows a report per project, and deletes them only when you add `--delete`.

```bash
devgc ~/Projects                # report only, nothing is deleted
devgc ~/Projects --delete       # delete (asks first)
devgc ~/Projects --deps         # also node_modules, Pods, vendor, .venv
devgc ~/Projects --only gradle,rust
```

## Install

```bash
go install github.com/kittayawat-aong/devgc@latest
```

## What it deletes

A folder is only matched when the file that proves what it is sits next to it,
so a folder named `build` is never deleted by its name alone.

| Type | Marker | Folders |
|---|---|---|
| gradle | `settings.gradle[.kts]` | `build`, `.gradle`, `.kotlin` |
| gradle | `build.gradle[.kts]` | `build`, `.cxx` |
| flutter | `pubspec.yaml` | `build`, `.dart_tool` |
| node | `package.json` | `.next`, `.nuxt`, `.turbo`, `.svelte-kit`, `.parcel-cache` |
| xcode | `*.xcodeproj` | `build`, `DerivedData` |
| cocoapods | `Podfile` | `build` |
| swift | `Package.swift` | `.build` |
| rust | `Cargo.toml` | `target` |
| maven | `pom.xml` | `target` |
| python | `pyproject.toml`, `requirements.txt` | `.pytest_cache`, `.mypy_cache`, `.ruff_cache` |

With `--deps`: `node_modules` (node), `Pods` (cocoapods), `vendor` (php),
`.venv` / `venv` (python). Reinstall them afterwards.

For node, `dist` and `build` are left alone because some projects commit them.

Folders are grouped by project: the nearest git repo (a worktree is its own project),
otherwise the topmost folder with a marker file. Symlinks are never followed.
Before deleting Gradle folders it stops running Gradle and Kotlin daemons.

## License

MIT
