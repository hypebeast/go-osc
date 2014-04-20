# GoOSC

This is an OSC protocol implementation in pure Go.

 * **Build Status:** [![Build Status](https://travis-ci.org/hypebeast/go-osc.png?branch=master)](https://travis-ci.org/hypebeast/go-osc)
 * **Documentation:** <http://godoc.org/github.com/hypebeast/go-osc/osc>

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
    ip := "localhost"
    port := 8765
 
    client := osc.NewOscClient(ip, port)
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
 
    server.ListenAndServe()
}
```
