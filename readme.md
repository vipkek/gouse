# `gouse`

Toggle ‘declared and not used’ errors in Go by using idiomatic `_ = notUsedVar`
and leaving a TODO comment. ![a demo](demo.gif)

## Installation

```sh
go install github.com/looshch/gouse@latest
```

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

`gouse` first removes previously created fake usages. If there is nothing to
remove, it builds the input and checks the build output for ‘declared and not
used’ errors. If there are any, it creates fake usages for the reported
unused variables.

## Integrations

- Vim: just bind `<cmd> w <bar> silent !gouse -w %<cr>` to some mapping.
- [Visual Studio Code plugin](https://marketplace.visualstudio.com/items?itemName=looshch.gouse).
- [Open VSX Registry plugin](https://open-vsx.org/extension/looshch/gouse)—works
  in Cursor, Windsurf, Google Antigravity, Kiro, Trae, Void.

## Credits

Inspired by [Nikita Rabaev](https://github.com/nikrabaev)’s idea.
