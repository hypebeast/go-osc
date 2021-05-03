# GoOSC

[![Build Status](https://travis-ci.org/hypebeast/go-osc.png?branch=master)](https://travis-ci.org/hypebeast/go-osc) [![GoDoc](https://godoc.org/github.com/hypebeast/go-osc/osc?status.svg)](https://godoc.org/github.com/hypebeast/go-osc/osc) [![Coverage Status](https://coveralls.io/repos/github/hypebeast/go-osc/badge.svg?branch=master)](https://coveralls.io/github/hypebeast/go-osc?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/hypebeast/go-osc)](https://goreportcard.com/report/github.com/hypebeast/go-osc)

[Open Sound Control (OSC)](http://opensoundcontrol.org/introduction-osc) library for Golang. Implemented in pure Go.

## Features

- OSC Bundles, including timetags
- OSC Messages
- OSC Client
- OSC Server
- Supports the following OSC argument types:
  - 'i' (Int32)
  - 'f' (Float32)
  - 's' (string)
  - 'b' (blob / binary data)
  - 'h' (Int64)
  - 't' (OSC timetag)
  - 'd' (Double/int64)
  - 'T' (True)
  - 'F' (False)
  - 'N' (Nil)
- Support for OSC address pattern including '\*', '?', '{,}' and '[]' wildcards

## Install

```shell
go get github.com/hypebeast/go-osc
```

## Usage

### Client

```go
package main

import "github.com/hypebeast/go-osc/osc"

func main() {
    client := osc.NewClient("localhost", 8765)
    defer client.Close()
    msg := osc.NewMessage("/osc/address")
    msg.Append(int32(111))
    msg.Append(true)
    msg.Append("hello")
    client.Send(msg)
}
```

### Client with response handling

```go
package main

import "github.com/hypebeast/go-osc/osc"

func main() {
    finished := make(chan struct{})

    client := osc.NewClient("localhost", 8765)
    defer client.Close()
    
    d := osc.NewStandardDispatcher()
    d.AddMsgHandler("/reply", func(msg *osc.Message) {
        osc.PrintMessage(msg)
        finished <- struct{}{}
    })
    client.SetDispatcher(d)
    go client.ListenAndServe()

    msg := osc.NewMessage("/message/address")
    client.Send(msg)

    <-finished
}
```

### Server

```go
package main

import "github.com/hypebeast/go-osc/osc"

func main() {
    var server *osc.Server
    addr := "127.0.0.1:8765"
    d := osc.NewStandardDispatcher()
    d.AddMsgHandler("/message/address", func(msg *osc.Message) {
        osc.PrintMessage(msg)
        server.SendTo(osc.NewMessage("/reply"), msg.SenderAddr())
    })

    server = &osc.Server{
        Addr: addr,
        Dispatcher:d,
    }
    server.ListenAndServe()
}
```

## Tests

```shell
make test
```
