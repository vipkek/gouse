# `gouse`

Toggle ‘declared and not used’ errors in Go by using idiomatic `_ = notUsedVar`
and leaving a TODO comment. ![a demo](demo.gif)

## Installation

```sh
go install github.com/looshch/gouse@latest
```

<!-- prettier-ignore -->
> [!CAUTION]
>
> Archived Go versions require older `gouse` version.
>
> | Go version   | gouse version |
> | ------------ |-------------- |
> | 1.21.0+      | 1.3.0+        |
> | 1.18–1.20.14 | 1.2.3         |

## Usage

By default, `gouse` accepts code from stdin or from a file provided as a path
argument and writes the toggled version to stdout. ‘-w’ flag writes the result
back to the file. If multiple paths provided, ‘-w’ flag is required.

### Examples

```sh
$ gouse
...input...
notUsed = true
...input...

...output...
notUsed = true; _ = notUsed /* TODO: gouse */
...output...
```

```sh
$ gouse main.go
...
notUsed = true; _ = notUsed /* TODO: gouse */
...
```

```sh
$ gouse -w main.go io.go core.go
$ cat main.go io.go core.go
...
notUsedFromMain = true; _ = notUsedFromMain /* TODO: gouse */
...
notUsedFromIo = true; _ = notUsedFromIo /* TODO: gouse */
...
notUsedFromCore = true; _ = notUsedFromCore /* TODO: gouse */
...
```

## How it works

First it tries to remove previously created fake usages. If there is nothing to
remove, it tries to build an input and checks the build stdout for ‘declared and
not used’ errors. If there is any, it creates fake usages for unused variables
from the errors.

## Integrations

- Vim: just bind `<cmd> w <bar> silent !gouse -w %<cr>` to some mapping.
- [Visual Studio Code plugin](https://marketplace.visualstudio.com/items?itemName=looshch.gouse).
- [Open VSX Registry plugin](https://open-vsx.org/extension/looshch/gouse)—works
  in Cursor, Windsurf, Google Antigravity, Kiro, Trae, Void.

## Credits

Inspired by [Nikita Rabaev](https://github.com/nikrabaev)’s idea.
