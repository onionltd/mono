package oniontree

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/go-yaml/yaml"
	"github.com/onionltd/oniontree-tools/pkg/types/service"
	"github.com/onionltd/oniontree-tools/pkg/types/tag"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	cairnName string = ".oniontree"
)

var (
	ErrCairnNotFound = errors.New("not an oniontree repository")
	ErrIdExists      = errors.New("id exists")
	ErrIdNotExists   = errors.New("id not exists")
)

type OnionTree struct {
	dir    string
	format string
}

func (o OnionTree) Init() error {
	for _, dir := range []string{o.getTaggedDir(), o.getUnsortedDir()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	pth := path.Join(o.dir, cairnName)
	cairnFile, err := os.Create(pth)
	if err != nil {
		return err
	}
	return cairnFile.Close()
}

func (o OnionTree) Add(id string, s service.Service) error {
	pth := path.Join(o.getUnsortedDir(), o.idToFilename(id))
	if isFile(pth) {
		return ErrIdExists
	}
	file, err := os.Create(pth)
	if err != nil {
		return err
	}
	data, err := o.marshalData(s)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Close()
}

func (o OnionTree) Edit(id string, s service.Service) error {
	pth := path.Join(o.getUnsortedDir(), o.idToFilename(id))
	if !isFile(pth) {
		return ErrIdNotExists
	}
	file, err := os.Create(pth)
	if err != nil {
		return err
	}
	data, err := o.marshalData(s)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Close()
}

func (o OnionTree) Tag(id string, tags []string) error {
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		pth := path.Join(o.getUnsortedDir(), o.idToFilename(id))
		if !isFile(pth) {
			return ErrIdNotExists
		}
		pthTag := path.Join(o.getTaggedDir(), tag)
		// Create tag directory, ignore error if it already exists.
		if err := os.Mkdir(pthTag, 0755); err != nil {
			if !os.IsExist(err) {
				return err
			}
		}
		pthRel, err := filepath.Rel(pthTag, pth)
		if err != nil {
			return err
		}
		if err := os.Symlink(pthRel, path.Join(pthTag, path.Base(pth))); err != nil {
			if !os.IsExist(err) {
				return err
			}
		}
	}
	return nil
}

func (o OnionTree) Get(id string) (service.Service, error) {
	data, err := o.GetRaw(id)
	if err != nil {
		return service.Service{}, err
	}
	s := service.Service{ID: id}
	if err := o.unmarshalData(data, &s); err != nil {
		return service.Service{}, err
	}
	return s, nil
}

func (o OnionTree) GetRaw(id string) ([]byte, error) {
	pth := path.Join(o.getUnsortedDir(), o.idToFilename(id))
	if !isFile(pth) {
		return nil, ErrIdNotExists
	}
	file, err := os.Open(pth)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data := []byte{}
	buff := make([]byte, 15535)
	for {
		num, err := file.Read(buff)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		data = append(data, buff[:num]...)
	}
	return data, nil
}

func (o OnionTree) GetTag(id string) (tag.Tag, error) {
	pth := path.Join(o.getTaggedDir(), id)
	if !isDir(pth) {
		return tag.Tag{}, ErrIdNotExists
	}
	t := tag.Tag{ID: id}
	file, err := os.Open(pth)
	if err != nil {
		return tag.Tag{}, err
	}
	defer file.Close()
	t.Services, err = file.Readdirnames(0)
	if err != nil {
		return tag.Tag{}, err
	}
	for idx, _ := range t.Services {
		t.Services[idx] = o.filenameToId(t.Services[idx])
	}
	sort.Strings(t.Services)
	return t, nil
}

func (o OnionTree) List() ([]string, error) {
	file, err := os.Open(o.getUnsortedDir())
	if err != nil {
		return nil, err
	}
	defer file.Close()
	files, err := file.Readdirnames(0)
	if err != nil {
		return nil, err
	}
	for idx, _ := range files {
		files[idx] = o.filenameToId(files[idx])
	}
	sort.Strings(files)
	return files, nil
}

func (o OnionTree) ListTags() ([]string, error) {
	file, err := os.Open(o.getTaggedDir())
	if err != nil {
		return nil, err
	}
	defer file.Close()
	files, err := file.Readdirnames(0)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// Hash calculates sha256 sum of OnionTree content.
// Services are read in alphabetical order and hash of their content is appended to a buffer.
// Resulting hash is sha256 sum of all hashes.
func (o OnionTree) Hash() ([32]byte, error) {
	services, err := o.List()
	if err != nil {
		return [32]byte{}, err
	}
	payload := make([]byte, len(services)*sha256.Size)
	for idx := range services {
		b, err := o.GetRaw(services[idx])
		if err != nil {
			return [32]byte{}, err
		}
		hash := sha256.Sum256(b)
		for i := range hash {
			payload[(idx*sha256.Size)+i] = hash[i]
		}
	}
	return sha256.Sum256(payload), nil
}

func (o OnionTree) getUnsortedDir() string {
	return path.Join(o.dir, "unsorted")
}

func (o OnionTree) getTaggedDir() string {
	return path.Join(o.dir, "tagged")
}

func (o OnionTree) marshalData(data interface{}) (b []byte, err error) {
	switch o.format {
	case "yaml":
		b, err = yaml.Marshal(data)
	default:
		panic("unsupported format")
	}
	return
}

func (o OnionTree) unmarshalData(b []byte, data interface{}) (err error) {
	switch o.format {
	case "yaml":
		err = yaml.Unmarshal(b, data)
	default:
		panic("unsupported format")
	}
	return
}

func (o OnionTree) idToFilename(id string) string {
	return fmt.Sprintf("%s.%s", id, o.format)
}

func (o OnionTree) filenameToId(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

func (o OnionTree) findRootDir(dir string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		pth := path.Join(dir, cairnName)
		match, err := filepath.Glob(pth)
		if err != nil {
			return "", err
		}
		if len(match) > 0 {
			relpth, err := filepath.Rel(cwd, path.Dir(pth))
			if err != nil {
				return "", err
			}
			return relpth, nil
		}
		if dir == "/" {
			break
		}
		dir = path.Join(dir, "..")
	}
	return "", ErrCairnNotFound
}

func New(dir string) *OnionTree {
	return &OnionTree{
		dir:    dir,
		format: "yaml",
	}
}

func Open(dir string) (*OnionTree, error) {
	o := &OnionTree{format: "yaml"}
	root, err := o.findRootDir(dir)
	if err != nil {
		return nil, err
	}
	o.dir = root
	return o, nil
}
