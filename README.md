# GoOSC

Open Sound Control (OSC) library for Golang. Implemented in pure Go.

 * **Build Status:** [![Build Status](https://travis-ci.org/hypebeast/go-osc.png?branch=master)](https://travis-ci.org/hypebeast/go-osc)
 * **Documentation:** <http://godoc.org/github.com/hypebeast/go-osc/osc>

[![views](https://sourcegraph.com/api/repos/github.com/hypebeast/go-osc/.counters/views.svg)](https://sourcegraph.com/github.com/hypebeast/go-osc)
[![views 24h](https://sourcegraph.com/api/repos/github.com/hypebeast/go-osc/.counters/views-24h.svg?no-count=1)](https://sourcegraph.com/github.com/hypebeast/go-osc)

[![library users](https://sourcegraph.com/api/repos/github.com/hypebeast/go-osc/.badges/library-users.svg)](https://sourcegraph.com/github.com/hypebeast/go-osc)
[![dependents](https://sourcegraph.com/api/repos/github.com/hypebeast/go-osc/.badges/dependents.svg)](https://sourcegraph.com/github.com/hypebeast/go-osc)

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
  * Support for OSC address pattern including '*', '?', '{,}' and '[]' wildcards

## Usage

### Client

```go
import "github.com/hypebeast/go-osc/osc"

func main() {
    client := osc.NewOscClient("localhost", 8765)
    msg := osc.NewOscMessage("/osc/address")
    msg.Append(int32(111))
    msg.Append(true)
    msg.Append("hello")
    client.Send(msg)
}
```

### Server

```go
import "github.com/hypebeast/go-osc/osc"

func main() {
    address := "127.0.0.1"
    port := 8765
    server := osc.NewOscServer(address, port)
 
    server.AddMsgHandler("/osc/address", func(msg *osc.OscMessage) {
        osc.PrintOscMessage(msg)
    })
 
    server.ListenAndDispatch()
}
```
