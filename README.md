# GoOSC

[Open Sound Control (OSC)](http://opensoundcontrol.org/introduction-osc) library for Golang. Implemented in pure Go.

* Build Status:  [![Build Status][CIStatus]][CIProject]
* Documentation: [![GoDoc][GoDocStatus]][GoDoc]
* Views:         [![Views][SGViews]][SGProject] [![views_24h][SGViews24h]][SGProject]
* Users:         [![library users][SGUsers]][SGProject] [![dependents][SGDependents]][SGProject]

## Features
  * OSC Bundles, including timetags
  * OSC Messages
  * OSC Client
  * OSC Server
  * Supports the following OSC argument types:
    * 'i' (Int32)
    * 'f' (Float32)
    * 's' (string)
    * 'b' (blob / binary data)
    * 'h' (Int64)
    * 't' (OSC timetag)
    * 'd' (Double/int64)
    * 'T' (True)
    * 'F' (False)
    * 'N' (Nil)
  * Support for OSC address pattern including '\*', '?', '{,}' and '[]' wildcards

## Usage

### Client

```go
import osc "github.com/kward/go-osc"

func main() {
    client := osc.NewClient("localhost", 8765)
    msg := osc.NewMessage("/osc/address")
    msg.Append(int32(111))
    msg.Append(true)
    msg.Append("hello")
    client.Send(msg)
}
```

### Server

```go
package main

import osc "github.com/kward/go-osc"

func main() {
  addr := "127.0.0.1:8765"
  server := &osc.Server{Addr: addr}

  server.Handle("/message/address", func(msg *osc.Message) {
    osc.PrintMessage(msg)
  })

  server.ListenAndServe()
}
```

## Misc
This library was forked from https://github.com/hypebeast/go-osc so that the Travis continuous integration could be fixed, enabling it to be imported into other projects that use Travis CI.


<!--- Links -->

[CIProject]: https://travis-ci.org/kward/go-osc
[CIStatus]: https://travis-ci.org/kward/go-osc.png?branch=master

[GoDoc]: https://godoc.org/github.com/kward/go-osc
[GoDocStatus]: https://godoc.org/github.com/kward/go-osc?status.svg

[SGProject]: https://sourcegraph.com/github.com/kward/go-osc
[SGDependents]: https://sourcegraph.com/api/repos/github.com/kward/go-osc/.badges/dependents.svg
[SGUsers]: https://sourcegraph.com/api/repos/github.com/kward/go-osc/.badges/library-users.svg
[SGViews]: https://sourcegraph.com/api/repos/github.com/kward/go-osc/.counters/views.svg
[SGViews24h]: https://sourcegraph.com/api/repos/github.com/kward/go-osc/.counters/views-24h.svg?no-count=1
