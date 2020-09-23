package evtcache

import (
	"context"
	"github.com/oniontree-org/go-oniontree/scanner"
	"sync"
)

type Cache struct {
	sync.RWMutex
	// Format: addresses[serviceID][address] = status
	addresses map[string]map[string]scanner.Status
	// Format: services[address]serviceID
	services map[string]string
}

func (c *Cache) ReadEvents(ctx context.Context, inputCh <-chan scanner.Event) error {
	c.init()
	defer c.uninit()

	for {
		select {
		case event, more := <-inputCh:
			if !more {
				return nil
			}

			switch e := event.(type) {
			case scanner.ScanEvent:
				c.addAddress(e.ServiceID, e.URL, e.Status)

			case scanner.WorkerStopped:
				c.deleteAddress(e.ServiceID, e.URL)

			case scanner.ProcessStopped:
				c.deleteService(e.ServiceID)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Cache) GetServiceID(url string) (string, bool) {
	return c.getServiceID(url)
}

func (c *Cache) GetAddresses(serviceID string) (map[string]scanner.Status, bool) {
	return c.getAddresses(serviceID)
}

func (c *Cache) GetOnlineAddresses(serviceID string) ([]string, bool) {
	addrs, ok := c.getAddresses(serviceID)
	online := make([]string, 0, len(addrs))
	for addr, status := range addrs {
		if status == scanner.StatusOffline {
			continue
		}
		online = append(online, addr)
	}
	return online, ok
}

func (c *Cache) init() {
	c.Lock()
	c.addresses = make(map[string]map[string]scanner.Status)
	c.services = make(map[string]string)
	c.Unlock()
}

func (c *Cache) uninit() {
	c.Lock()
	c.addresses = nil
	c.services = nil
	c.Unlock()
}

func (c *Cache) deleteAddress(serviceID, address string) {
	c.Lock()
	if _, ok := c.addresses[serviceID]; ok {
		delete(c.addresses[serviceID], address)
	}
	delete(c.services, address)
	c.Unlock()
}

func (c *Cache) deleteService(serviceID string) {
	c.Lock()
	delete(c.addresses, serviceID)
	c.Unlock()
}

func (c *Cache) addAddress(serviceID, address string, status scanner.Status) {
	c.Lock()
	if _, ok := c.addresses[serviceID]; !ok {
		c.addresses[serviceID] = make(map[string]scanner.Status)
	}
	c.addresses[serviceID][address] = status
	c.services[address] = serviceID
	c.Unlock()
}

func (c *Cache) getAddresses(serviceID string) (map[string]scanner.Status, bool) {
	c.RLock()
	defer c.RUnlock()
	v, ok := c.addresses[serviceID]
	return v, ok
}

func (c *Cache) getServiceID(url string) (string, bool) {
	c.RLock()
	defer c.RUnlock()
	v, ok := c.services[url]
	return v, ok
}
