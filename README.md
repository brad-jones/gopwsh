# gopwsh

[![PkgGoDev](https://pkg.go.dev/badge/github.com/brad-jones/gopwsh)](https://pkg.go.dev/github.com/brad-jones/gopwsh)
[![GoReport](https://goreportcard.com/badge/github.com/brad-jones/gopwsh)](https://goreportcard.com/report/github.com/brad-jones/gopwsh)
[![GoLang](https://img.shields.io/badge/golang-%3E%3D%201.15.1-lightblue.svg)](https://golang.org)
![.github/workflows/main.yml](https://github.com/brad-jones/gopwsh/workflows/.github/workflows/main.yml/badge.svg?branch=master)
[![semantic-release](https://img.shields.io/badge/%20%20%F0%9F%93%A6%F0%9F%9A%80-semantic--release-e10079.svg)](https://github.com/semantic-release/semantic-release)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://conventionalcommits.org)
[![KeepAChangelog](https://img.shields.io/badge/Keep%20A%20Changelog-1.0.0-%23E05735)](https://keepachangelog.com/)
[![License](https://img.shields.io/github/license/brad-jones/gopwsh.svg)](https://github.com/brad-jones/gopwsh/blob/master/LICENSE)

Package gopwsh is a simple host for PowerShell with-in your Go code.

Originally inspired by <https://github.com/bhendo/go-powershell>

## Quick Start

`go get -u github.com/brad-jones/gopwsh`

```go
package main

import (
	"fmt"

	"github.com/brad-jones/gopwsh"
)

func main() {
	shell := gopwsh.MustNew()
	defer shell.Exit()
	stdout, _, err := shell.Execute("Get-ComputerInfo")
	if err != nil {
		panic(err)
	}
	fmt.Println(stdout)
}
```

_Also see further working examples under: <https://github.com/brad-jones/gopwsh/tree/master/examples>_

## Cross Platform Support

PowerShell these days of course is a cross platform shell,
able to run on Windows, MacSO & Linux.

This go module should in theory work on all those platforms too.

But _(at this stage)_ has not been tested anywhere apart from Windows.
Who uses PowerShell outside of Windows... sorry thats short sighted & naive of me.

Eventually I'll get around to writing a full test suite but until then if you
are one of these users & notice a bug, PRs are of course welcome :)
