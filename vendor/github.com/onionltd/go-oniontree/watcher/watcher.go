package watcher

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/onionltd/go-oniontree"
	"path"
	"path/filepath"
	"strings"
)

type Watcher struct {
	ot *oniontree.OnionTree
}

func (w *Watcher) watchTagged(
	ctx context.Context,
	eventCh chan<- Event,
) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recovered: %s\n", r)
		}
	}()
	filenameToTagName := func(f string) string {
		return path.Base(f)
	}
	fsEventToTagEvent := func(e fsnotify.Event) Event {
		tagName := filenameToTagName(e.Name)
		switch e.Op {
		case fsnotify.Create:
			return tagCreated{
				Name: tagName,
			}
		}
		return nil
	}
	emitEvent := func(e Event) {
		select {
		case eventCh <- e:
		}
	}

	taggedWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer taggedWatcher.Close()
	if err := taggedWatcher.Add(w.ot.TaggedDir()); err != nil {
		return err
	}

	for {
		select {
		case e := <-taggedWatcher.Events:
			if event := fsEventToTagEvent(e); event != nil {
				emitEvent(event)
			}

		case err := <-taggedWatcher.Errors:
			return err

		case <-ctx.Done():
			return nil
		}
	}
}

func (w *Watcher) watchTags(
	ctx context.Context,
	taggedEventCh <-chan Event,
	eventCh chan<- Event,
) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recovered: %s\n", r)
		}
	}()
	emitEvent := func(e Event) {
		select {
		case eventCh <- e:
		}
	}
	filenameToServiceID := func(f string) string {
		f = path.Base(f)
		return strings.TrimSuffix(f, filepath.Ext(f))
	}
	filenameToTagName := func(f string) string {
		return path.Base(path.Dir(f))
	}
	fsEventToServiceEvent := func(e fsnotify.Event) Event {
		tagName := filenameToTagName(e.Name)
		serviceID := filenameToServiceID(e.Name)
		// If tag name is named "tagged", it means the tag directory itself
		// was removed. Such event must not be forwarded.
		if tagName == "tagged" {
			return nil
		}
		switch e.Op {
		case fsnotify.Create:
			return ServiceTagged{
				ID:  serviceID,
				Tag: tagName,
			}
		case fsnotify.Remove:
			return ServiceUntagged{
				ID:  serviceID,
				Tag: tagName,
			}
		}
		return nil
	}

	tagsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer tagsWatcher.Close()

	tags, err := w.ot.ListTags()
	if err != nil {
		return err
	}

	for i := range tags {
		pth := path.Join(w.ot.TaggedDir(), tags[i])
		if err := tagsWatcher.Add(pth); err != nil {
			return err
		}
	}

	for {
		select {
		case e := <-taggedEventCh:
			switch t := e.(type) {
			case tagCreated:
				// WARNING: There's a lingering race condition!
				//
				// If a service is tagged with a tag that didn't exist before,
				// a new directory `tagged/{foo}` is created, followed by a symbolic link
				// `tagged/{foo}/{bar}.yaml`. The newly created directory is added to the watch list,
				// however, the symbolic link may be created long before that happens, and create
				// event gets lost.
				//
				// A duct-tape solution is to look into the directory immediately after it's created
				// and treat all the files found there as a new tag and emit the event for each one of them.
				// Only after that add the directory to the watch list. This doesn't prevent the race
				// condition but lowers the likelihood.
				//
				// The race condition may manifest itself as:
				//
				// * Emitted duplicate ServiceTagged events.
				// * Lost ServiceTagged event.
				serviceIDs, err := w.ot.ListServicesWithTag(t.Name)
				if err != nil {
					return err
				}

				for i := range serviceIDs {
					emitEvent(ServiceTagged{
						ID:  serviceIDs[i],
						Tag: t.Name,
					})
				}

				// Start watching newly created tag directory
				pth := path.Join(w.ot.TaggedDir(), t.Name)
				if err := tagsWatcher.Add(pth); err != nil {
					return err
				}
			}

		case e := <-tagsWatcher.Events:
			if event := fsEventToServiceEvent(e); event != nil {
				emitEvent(event)
			}

		case err := <-tagsWatcher.Errors:
			return err

		case <-ctx.Done():
			return nil
		}
	}
}

func (w *Watcher) watchUnsorted(
	ctx context.Context,
	eventCh chan<- Event,
) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recovered: %s\n", r)
		}
	}()
	emitEvent := func(e Event) {
		select {
		case eventCh <- e:
		}
	}
	filenameToServiceID := func(f string) string {
		f = path.Base(f)
		return strings.TrimSuffix(f, filepath.Ext(f))
	}
	fsEventToServiceEvent := func(e fsnotify.Event) Event {
		serviceID := filenameToServiceID(e.Name)
		switch e.Op {
		case fsnotify.Create:
			return ServiceAdded{
				ID: serviceID,
			}
		case fsnotify.Remove:
			return ServiceRemoved{
				ID: serviceID,
			}
		case fsnotify.Write:
			return ServiceUpdated{
				ID: serviceID,
			}
		}
		return nil
	}

	unsortedWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer unsortedWatcher.Close()
	if err := unsortedWatcher.Add(w.ot.UnsortedDir()); err != nil {
		return err
	}

	for {
		select {
		case e := <-unsortedWatcher.Events:
			if event := fsEventToServiceEvent(e); event != nil {
				emitEvent(event)
			}

		case err := <-unsortedWatcher.Errors:
			return err

		case <-ctx.Done():
			return nil
		}
	}
}

// Watch watches for events, emitting them to channel `eventCh`.
// The call to Watch blocks until there's an error, or the context `ctx` is canceled.
func (w *Watcher) Watch(
	ctx context.Context,
	eventCh chan<- Event,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer close(eventCh)

	errCh := make(chan error, 3)
	taggedEventCh := make(chan Event)

	go func() {
		if err := w.watchTagged(ctx, taggedEventCh); err != nil {
			errCh <- err
		}
	}()

	go func() {
		if err := w.watchTags(ctx, taggedEventCh, eventCh); err != nil {
			errCh <- err
		}
	}()

	go func() {
		if err := w.watchUnsorted(ctx, eventCh); err != nil {
			errCh <- err
		}
	}()

	return <-errCh
}

// NewWatcher returns a new Watcher.
func NewWatcher(ot *oniontree.OnionTree) *Watcher {
	return &Watcher{
		ot: ot,
	}
}
