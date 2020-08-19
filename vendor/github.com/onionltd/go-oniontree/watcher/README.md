# Watcher

Watcher is a package leveraging [fsnotify](https://github.com/fsnotify/fsnotify)
to watch events in an OnionTree repository and emitting them to a channel.

## Example

```go
package main

import (
    "fmt"
    "context"
    "github.com/onionltd/go-oniontree"
    "github.com/onionltd/go-oniontree/watcher"
)

func main() {
    ot, err := oniontree.Open(".")
    if err != nil {
        panic(err)
    }
    w := watcher.NewWatcher(ot)

    eventCh := make(chan watcher.Event)

    go func(){
        if err := w.Watch(context.TODO(), eventCh); err != nil {
            panic(err)
        }
    }()

    for {
        select {
        case e := <-eventCh:
            switch e.(type) {
            case watcher.ServiceAdded:
                fmt.Println("service added!")
            case watcher.ServiceRemoved:
                fmt.Println("service removed!")
            }
        }
    }
}
```
