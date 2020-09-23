package oniontree

import (
	"fmt"
	"github.com/go-yaml/yaml"
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

type OnionTree struct {
	dir    string
	format string
}

// Init initializes empty repository.
func (o OnionTree) Init() error {
	for _, dir := range []string{o.TaggedDir(), o.UnsortedDir()} {
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

// Add adds a new service to the repository with data from `s`.
func (o OnionTree) AddService(s *Service) error {
	if err := s.Validate(); err != nil {
		return err
	}
	pth := path.Join(o.UnsortedDir(), o.idToFilename(s.ID()))
	file, err := os.OpenFile(pth, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		if os.IsExist(err) {
			return &ErrIdExists{s.ID()}
		}
		return err
	}
	defer file.Close()
	data, err := o.marshalData(s)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	return nil
}

// Remove removes a service `id` from the repository with all its tags.
func (o OnionTree) RemoveService(id string) error {
	tags, err := o.ListServiceTags(id)
	if err != nil {
		return err
	}
	if err := o.UntagService(id, tags); err != nil {
		return err
	}
	pth := path.Join(o.UnsortedDir(), o.idToFilename(id))
	if err := os.Remove(pth); err != nil {
		if os.IsNotExist(err) {
			return &ErrIdNotExists{id}
		}
		return err
	}
	return nil
}

// Update replaces existing service with new data from `s`.
func (o OnionTree) UpdateService(s *Service) error {
	if err := s.Validate(); err != nil {
		return err
	}
	pth := path.Join(o.UnsortedDir(), o.idToFilename(s.ID()))
	file, err := os.OpenFile(pth, os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			return &ErrIdNotExists{s.ID()}
		}
		return err
	}
	defer file.Close()
	data, err := o.marshalData(s)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	return nil
}

// GetService returns content of service `id`.
func (o OnionTree) GetService(id string) (*Service, error) {
	data, err := o.GetServiceBytes(id)
	if err != nil {
		return nil, err
	}
	s := NewService(id)
	if err := o.unmarshalData(data, &s); err != nil {
		return nil, err
	}
	return s, nil
}

// GetServiceBytes returns raw bytes of service `id`.
func (o OnionTree) GetServiceBytes(id string) ([]byte, error) {
	pth := path.Join(o.UnsortedDir(), o.idToFilename(id))
	file, err := os.Open(pth)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ErrIdNotExists{id}
		}
		return nil, err
	}
	defer file.Close()
	data := []byte{}
	buff := make([]byte, 1024)
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

// ListServices returns a list of service IDs found in the repository.
func (o OnionTree) ListServices() ([]string, error) {
	file, err := os.Open(o.UnsortedDir())
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

// ListServicesWithTag returns a list of services tagged with `tag`.
func (o OnionTree) ListServicesWithTag(tag Tag) ([]string, error) {
	pth := path.Join(o.TaggedDir(), tag.String())
	file, err := os.Open(pth)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ErrTagNotExists{tag}
		}
		return nil, err
	}
	defer file.Close()
	services, err := file.Readdirnames(0)
	if err != nil {
		return nil, err
	}
	for idx, _ := range services {
		services[idx] = o.filenameToId(services[idx])
	}
	sort.Strings(services)
	return services, nil
}

// ListTags returns a list of tags found in the repository.
func (o OnionTree) ListTags() ([]Tag, error) {
	file, err := os.Open(o.TaggedDir())
	if err != nil {
		return nil, err
	}
	defer file.Close()
	files, err := file.Readdirnames(0)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	tags := make([]Tag, len(files))
	for i := range files {
		tags[i] = Tag(files[i])
	}
	return tags, nil
}

// ListServiceTags returns tags of service `id`.
// NOTICE: This function is very inefficient as it has to scale down the
// tagged directory recursively to find all symbolic links matching a pattern.
func (o OnionTree) ListServiceTags(id string) ([]Tag, error) {
	matching := []Tag{}
	tags, err := o.ListTags()
	if err != nil {
		return nil, err
	}
	for i := range tags {
		serviceIDs, err := o.ListServicesWithTag(tags[i])
		if err != nil {
			return nil, err
		}
		for _, serviceID := range serviceIDs {
			if serviceID == id {
				matching = append(matching, tags[i])
				break
			}
		}
	}
	return matching, nil
}

// TagService adds tags `tags` to service `id`.
func (o OnionTree) TagService(id string, tags []Tag) error {
	for _, tag := range tags {
		if err := tag.Validate(); err != nil {
			return err
		}
		filename := o.idToFilename(id)
		pth := path.Join(o.UnsortedDir(), filename)
		if !isFile(pth) {
			return &ErrIdNotExists{id}
		}
		pthTag := path.Join(o.TaggedDir(), tag.String())
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
		// Create tag, ignore error if it already exists.
		if err := os.Symlink(pthRel, path.Join(pthTag, filename)); err != nil {
			if !os.IsExist(err) {
				return err
			}
		}
	}
	return nil
}

// UntagService removes tags `tags` from service `id`.
func (o OnionTree) UntagService(id string, tags []Tag) error {
	for _, tag := range tags {
		filename := o.idToFilename(id)
		pth := path.Join(o.UnsortedDir(), filename)
		if !isFile(pth) {
			return &ErrIdNotExists{id}
		}
		pthTag := path.Join(o.TaggedDir(), tag.String())
		pthLink := path.Join(pthTag, filename)
		if isSymlink(pthLink) {
			if err := os.Remove(pthLink); err != nil {
				return err
			}
		}
		if isEmptyDir(pthTag) {
			if err := os.Remove(pthTag); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o OnionTree) Dir() string {
	return o.dir
}

func (o OnionTree) UnsortedDir() string {
	return path.Join(o.dir, "unsorted")
}

func (o OnionTree) TaggedDir() string {
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

const maxDepth = 8

func (o OnionTree) findRootDir(dir string) (string, error) {
	for i := 0; i < maxDepth; i++ {
		// TODO: It would be better to return an error from isDir (if there'd be one).
		if !isDir(dir) {
			return "", &ErrNotOnionTree{dir}
		}

		pth := path.Join(dir, cairnName)

		if isFile(pth) {
			return path.Dir(pth), nil
		}

		dir = path.Join(dir, "..")
	}
	return "", &ErrNotOnionTree{dir}
}

// New returns initialized OnionTree structure. The function
// does not check if `dir` is a valid OnionTree repository.
func New(dir string) *OnionTree {
	return &OnionTree{
		dir:    dir,
		format: "yaml",
	}
}

// Open attempts to "open" `dir` as a valid OnionTree repository.
// The function fails if the `dir` is not a valid OnionTree repository.
func Open(dir string) (*OnionTree, error) {
	o := &OnionTree{format: "yaml"}
	root, err := o.findRootDir(dir)
	if err != nil {
		return nil, err
	}
	o.dir = root
	return o, nil
}
