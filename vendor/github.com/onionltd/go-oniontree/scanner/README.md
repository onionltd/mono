# Scanner

Scanner is a concurrent, configurable TCP scanner for OnionTree content.

## Example

```go
package main

import (
    "fmt"
    "context"
    "github.com/onionltd/go-oniontree"
    "github.com/onionltd/go-oniontree/scanner"
)

func main() {
    s := scanner.NewScanner(scanner.DefaultScannerConfig)

    eventCh := make(chan scanner.Event)

    go func(){
        if err := s.Start(context.TODO(), ".", eventCh); err != nil {
            panic(err)
        }
    }()

    for {
        select {
        case e := <-eventCh:
            switch event := e.(type) {
            case scanner.ScanEvent:
                fmt.Printf("%+v\n", event)
            }
        }
    }
}
```
