Golang Template  [![Travis-CI](https://travis-ci.org/cloudfly/template.svg)](https://travis-ci.org/cloudfly/template) [![GoDoc](https://godoc.org/github.com/cloudfly/template?status.svg)](http://godoc.org/github.com/cloudfly/template) [![Go Report Card](https://goreportcard.com/badge/github.com/cloudfly/template)](https://goreportcard.com/report/github.com/cloudfly/template)
====

A template which is used to get a golang value by template string from a context. Instead of parsing template into a bytes stream, it return the golang value directlly.

## Installation

```bash
go get github.com/cloudfly/template
```

## Example

```go
package main

import (
        "fmt"
        "github.com/cloudfly/template"
)

type Context struct {
	data map[string]interface{}
}

func (c Context) Value(key string) interface{} {
	if d, ok := c.data[key]; ok {
		return d
	}
	return nil
}

func main() {
	ctx := Context{
		data: map[string]interface{}{
                    "b": true,
                },
	}

        result, err := template.Parse("{{ .b }}", ctx)
        if err != nil {
                panic(err)
        }
        // result is not a string or bytes, it's a interface{} of a boolean.
        fmt.Println(result.(bool))
        // print true
}
```


## Contributting

It parse the template by using [text/template/parse](http://golang.org/pkg/text/template/parse), and the realization of it also refrences the [text/template](http://golang.org/pkg/text/template) package.

The template syntax is same with `text/template`, but only support simple expression that returning a golang value, no range, no if-else, no block.
