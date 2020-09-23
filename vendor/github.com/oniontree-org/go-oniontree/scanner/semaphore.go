package scanner

import "golang.org/x/sync/semaphore"

// workerConnSem is used to limit number of simultaneous outbound TCP connections.
var workerConnSem *semaphore.Weighted
