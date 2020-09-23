package watcher

type Event interface{}

type (
	ServiceAdded struct {
		ID string
	}

	ServiceRemoved struct {
		ID string
	}

	ServiceUpdated struct {
		ID string
	}

	ServiceTagged struct {
		ID  string
		Tag string
	}

	ServiceUntagged struct {
		ID  string
		Tag string
	}
)

type (
	tagCreated struct {
		Name string
	}
)
