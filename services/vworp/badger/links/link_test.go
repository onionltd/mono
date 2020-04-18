package links_test

import (
	badger "github.com/dgraph-io/badger/v2"
	badgerutil "github.com/onionltd/mono/pkg/utils/badger"
	"github.com/onionltd/mono/services/vworp/badger/links"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestStore(t *testing.T) {
	const (
		serviceID = "example"
		url       = "/article/why-birds-flap-their-wings"
	)
	link, err := links.NewLink(serviceID, url)
	if err != nil {
		t.Fatal(err)
	}

	tempDir, err := ioutil.TempDir("/tmp", "vworp-ut")
	if err != nil {
		t.Fatal(err)
	}

	db, err := badger.Open(badger.DefaultOptions(tempDir))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Update(badgerutil.Store(link)); err != nil {
		t.Fatal(err)
	}

	readLink := &links.Link{}
	if err := db.View(badgerutil.Load(link.Key(), readLink)); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, readLink, link)
}

func TestNewLink(t *testing.T) {
	const (
		serviceID = "example"
		url       = "/article/why-birds-flap-their-wings"
	)
	_, err := links.NewLink(serviceID, url)
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkNewLink(b *testing.B) {
	const (
		serviceID = "example"
		url       = "/article/why-birds-flap-their-wings"
	)
	for n := 0; n < b.N; n++ {
		_, _ = links.NewLink(serviceID, url)
	}
}
