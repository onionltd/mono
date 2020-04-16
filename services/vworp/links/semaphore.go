package links

import "golang.org/x/sync/semaphore"

const connectionsMax int64 = 10

var connSem = semaphore.NewWeighted(connectionsMax)
