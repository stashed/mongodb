// Package crds Code generated by go-bindata. (@generated) DO NOT EDIT.
// sources:
// installer.stash.appscode.com_stashmongodbs.v1.yaml
// installer.stash.appscode.com_stashmongodbs.yaml
package crds

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// ModTime return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _installerStashAppscodeCom_stashmongodbsV1Yaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xb4\x55\x4d\x93\x1b\x45\x0c\xbd\xfb\x57\xa8\x8a\x43\x2e\x78\x5c\x4b\x2e\xd4\xdc\xc0\xe1\x90\x82\x00\x15\xa7\x72\x97\x7b\xe4\xb1\xd8\xfe\x42\x52\xbb\xb2\xfc\x7a\xaa\x7b\xc6\xbb\xb6\xd7\x6b\xc8\x16\xf4\xc9\xad\x8f\x37\xd2\x7b\x6a\x19\x33\x7f\x26\x51\x4e\xb1\x07\xcc\x4c\x5f\x8c\x62\xbd\x69\x77\xff\xbd\x76\x9c\x56\x87\xbb\xc5\x3d\xc7\xa1\x87\x75\x51\x4b\xe1\x23\x69\x2a\xe2\xe8\x1d\xed\x38\xb2\x71\x8a\x8b\x40\x86\x03\x1a\xf6\x0b\x00\x27\x84\xd5\xf8\x89\x03\xa9\x61\xc8\x3d\xc4\xe2\xfd\x02\xc0\xe3\x96\xbc\xd6\x18\x00\xcc\xb9\x07\x35\xd4\xfd\x02\x20\x62\xa0\xf9\x16\x52\x1c\xd3\xb0\xd5\x8e\xa3\x1a\x7a\x4f\xd2\x35\x7b\x87\x39\xab\x4b\x03\x75\x2e\x85\x85\x66\x72\x15\x67\x94\x54\x72\x0f\x37\x63\x27\xf8\xf9\xb3\x0e\x8d\xc6\x24\x7c\xbc\x2f\x1f\x6b\xa8\xbf\x8f\x79\xed\x3a\xb5\xbc\xa9\xee\x0f\xb5\xa8\x77\x3f\x36\xb3\x67\xb5\x9f\x9f\xb9\x7e\x61\xb5\xe6\xce\xbe\x08\xfa\x8b\x66\x9a\x47\x39\x8e\xc5\xa3\x9c\xfb\x16\x00\xea\x52\xa6\x1e\x7e\xad\x55\x66\x74\x34\x2c\x00\x0e\x93\x20\xad\xca\xe5\xcc\xcf\xe1\x0e\x7d\xde\xe3\xdd\x84\xe6\xf6\x14\x70\x6a\x02\x20\x65\x8a\x3f\xfc\xfe\xfe\xf3\xdb\xcd\x99\x19\x20\x4b\xca\x24\xf6\xd8\xef\x74\x4e\x14\x3f\xb1\x02\x0c\xa4\x4e\x38\x5b\x1b\x85\x37\x15\x70\x8a\x82\xa1\x4a\x4d\x0a\xb6\xa7\x63\x69\x34\xcc\x35\x40\xda\x81\xed\x59\x41\x28\x0b\x29\x45\x6b\xf2\x9f\x01\x43\x0d\xc2\x08\x69\xfb\x07\x39\xeb\x60\x43\x52\x61\x40\xf7\xa9\xf8\x01\x5c\x8a\x07\x12\x03\x21\x97\xc6\xc8\x7f\x3d\x62\x2b\x58\x6a\x1f\xf5\x68\x34\x33\xfc\x74\x38\x1a\x49\x44\x0f\x07\xf4\x85\xbe\x05\x8c\x03\x04\x7c\x00\xa1\xfa\x15\x28\xf1\x04\xaf\x85\x68\x07\x1f\x92\x10\x70\xdc\xa5\x1e\xf6\x66\x59\xfb\xd5\x6a\x64\x3b\x4e\xba\x4b\x21\x94\xc8\xf6\xb0\x72\x29\x9a\xf0\xb6\x58\x12\x5d\x0d\x74\x20\xbf\x52\x1e\x97\x28\x6e\xcf\x46\xce\x8a\xd0\x0a\x33\x2f\x5b\xe9\xd1\xda\x73\x09\xc3\x37\x32\xbf\x0d\x7d\x73\x56\xab\x3d\xe4\x36\xe0\xc2\x71\x3c\x71\xb4\x11\xbb\xa1\x40\x9d\x33\x60\x05\x9c\x53\xa7\x2e\x9e\x88\xae\xa6\xca\xce\xc7\x9f\x36\x9f\xe0\xf8\xe9\x26\xc6\x25\xfb\x8d\xf7\xa7\x44\x7d\x92\xa0\x12\xc6\x71\x47\x32\x89\xb8\x93\x14\x1a\x26\xc5\x21\x27\x8e\xd6\x2e\xce\x33\xc5\x4b\xfa\xb5\x6c\x03\x5b\xd5\xfd\xcf\x42\x6a\x55\xab\x0e\xd6\x18\x63\x32\xd8\x12\x94\x3c\xa0\xd1\xd0\xc1\xfb\x08\x6b\x0c\xe4\xd7\xa8\xf4\xbf\x0b\x50\x99\xd6\x65\x25\xf6\xdf\x49\x70\xba\xb9\x2e\x83\x27\xd6\x4e\x1c\xc7\xb5\xf3\x82\x5e\xa7\x0b\x61\x93\xc9\x55\xe9\x2a\x7b\xf3\x3b\xd9\x25\x99\x42\x60\x8e\x99\x87\x12\x76\xec\xe9\x0c\xf5\xfa\xab\xad\x67\x8b\xee\xbe\xe4\x4b\xeb\xad\x8c\x7a\x50\xc6\xab\xf6\x17\x59\xb9\xc9\x42\x3d\xbb\xe2\x7d\xdd\x4a\xbf\x1d\x48\x84\x07\x7a\x8e\x7e\x03\x99\x03\x8e\x57\x32\x6e\xf7\x20\x34\xb2\x9a\x3c\xbc\xa2\x8f\x9a\x9c\x93\xb2\xa5\x57\xa6\x1b\x8e\xaf\xc8\xab\xef\x82\x85\x86\xe7\xa9\xcb\xc7\x6e\xae\xba\x8e\xb5\x5e\x71\x1a\x7e\x95\x4c\x01\xbf\xac\x53\x74\x45\x84\xa2\xbb\xd2\xfb\x2e\x49\x40\xab\x7f\xa0\xf6\xf6\xbb\x17\x80\xeb\x96\x1d\x49\x2e\xbc\xaf\x16\x5f\x48\x2d\xc9\x57\xcb\xff\x5f\x8f\xf0\x75\x71\x96\xf3\x03\xbb\x30\xb6\x81\xbd\xb0\x9d\x53\x7b\xe1\x9c\x9b\xfc\xe7\x9d\xf2\xcc\xa8\x75\x2d\x0f\x3d\x98\x94\x29\xbd\x02\xd5\xe7\x32\x59\xfe\x0e\x00\x00\xff\xff\x13\x3a\x68\x12\xac\x09\x00\x00")

func installerStashAppscodeCom_stashmongodbsV1YamlBytes() ([]byte, error) {
	return bindataRead(
		_installerStashAppscodeCom_stashmongodbsV1Yaml,
		"installer.stash.appscode.com_stashmongodbs.v1.yaml",
	)
}

func installerStashAppscodeCom_stashmongodbsV1Yaml() (*asset, error) {
	bytes, err := installerStashAppscodeCom_stashmongodbsV1YamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "installer.stash.appscode.com_stashmongodbs.v1.yaml", size: 2476, mode: os.FileMode(420), modTime: time.Unix(1573722179, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _installerStashAppscodeCom_stashmongodbsYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xb4\x56\x5f\x6f\x1c\x35\x10\x7f\xbf\x4f\x31\x12\x48\x05\xc4\xee\x11\x2a\x21\xd8\x17\x04\x29\x48\x15\x2d\xa0\xa6\xf4\x25\x2a\xd2\x9c\x3d\xbb\x67\xe2\xb5\xcd\xcc\xec\x91\xe4\xd3\x23\x7b\x77\x2f\xf7\x37\x34\x08\xfc\x74\xfe\xcd\xff\xdf\xcc\xf8\x16\x93\x7b\x47\x2c\x2e\x86\x06\x30\x39\xba\x55\x0a\xf9\x26\xf5\xcd\xd7\x52\xbb\xb8\xdc\x5c\xac\x48\xf1\x62\x71\xe3\x82\x6d\xe0\x72\x10\x8d\xfd\x1b\x92\x38\xb0\xa1\x17\xd4\xba\xe0\xd4\xc5\xb0\xe8\x49\xd1\xa2\x62\xb3\x00\x30\x4c\x98\xc1\xb7\xae\x27\x51\xec\x53\x03\x61\xf0\x7e\x01\xe0\x71\x45\x5e\xb2\x0e\x00\xa6\xd4\x80\x28\xca\x7a\x01\x10\xb0\xa7\xe9\xd6\xc7\xd0\x45\xbb\x92\xda\x05\x51\xf4\x9e\xb8\x2e\x78\x8d\x29\x89\x89\x96\x6a\x13\xfb\x85\x24\x32\xd9\x4f\xc7\x71\x48\x0d\x3c\xaa\x3b\xba\x9f\xc2\x1a\x54\xea\x22\xbb\xf9\x5e\x6d\x73\xc8\xbf\x67\xbb\x72\x1d\x4b\xbe\xca\xe2\xd7\x39\xa9\x17\xdf\x17\xd8\x3b\xd1\x9f\x8e\x44\xaf\x9c\x68\x11\x27\x3f\x30\xfa\x83\x62\x8a\x44\x5c\xe8\x06\x8f\xbc\x2f\x5b\x00\x24\x26\x21\xde\xd0\x6f\xe1\x26\xc4\xbf\xc2\x8f\x8e\xbc\x95\x06\x5a\xf4\x92\x33\x11\x13\x13\x35\xf0\x73\x2e\x22\xa1\x21\xbb\x00\xd8\xa0\x77\xb6\xb0\x3c\x96\x11\x13\x85\xef\x7e\x7d\xf9\xee\xf9\x95\x59\x53\x8f\x23\x98\x3d\xc7\x44\xac\xdb\x6a\x47\xe2\xb7\x2d\xdf\x62\x00\x96\xc4\xb0\x4b\xc5\x23\x3c\xcb\xae\x46\x1d\xb0\xb9\xc9\x24\xa0\x6b\x82\xcd\x88\x91\x05\x29\x61\x20\xb6\xa0\x6b\x27\xc0\x54\x6a\x08\x5a\x52\xda\x71\x0b\x59\x05\x03\xc4\xd5\x1f\x64\xb4\x86\xab\x5c\x27\x0b\xc8\x3a\x0e\xde\x82\x89\x61\x43\xac\xc0\x64\x62\x17\xdc\xfd\xd6\xb3\x80\xc6\x12\xd2\xa3\xd2\xc4\xec\x7c\x5c\x50\xe2\x80\x3e\x93\x30\xd0\xe7\x80\xc1\x42\x8f\x77\xc0\x94\x63\xc0\x10\x76\xbc\x15\x15\xa9\xe1\x75\x64\x02\x17\xda\xd8\xc0\x5a\x35\x49\xb3\x5c\x76\x4e\xe7\x21\x37\xb1\xef\x87\xe0\xf4\x6e\x69\x62\x50\x76\xab\x41\x23\xcb\xd2\xd2\x86\xfc\x52\x5c\x57\x21\x9b\xb5\x53\x32\x3a\x30\x2d\x31\xb9\xaa\x24\x1e\xb4\x6c\x4a\x6f\x3f\xe2\x69\x23\xe4\xd9\x4e\xa6\x7a\x97\xca\x50\xb3\x0b\xdd\x16\x2e\x43\x75\x96\xf7\x3c\x57\xe0\x04\x70\x32\x1b\xf3\x7f\xa0\x37\x43\x99\x95\x37\x3f\x5c\xbd\x85\x39\x68\x69\xc1\x3e\xe7\x85\xed\x07\x33\x79\x20\x3e\x13\xe5\x42\x4b\x3c\x36\xae\xe5\xd8\x17\x8f\x14\x6c\x8a\x2e\x68\xb9\x18\xef\x28\xec\x93\x2e\xc3\xaa\x77\x9a\x3b\xfd\xe7\x40\xa2\xb9\x3f\x35\x5c\x62\x08\x51\x61\x45\x30\x24\x8b\x4a\xb6\x86\x97\x01\x2e\xb1\x27\x7f\x89\x42\xff\x3b\xed\x99\x61\xa9\x32\xa5\xff\x4c\xfc\xee\x0b\x35\x9f\x53\xeb\x91\x4f\x79\x8e\xf6\x10\x80\x1e\x6f\x5f\x51\xe8\x74\xdd\xc0\x57\xcf\x0f\x64\x09\x35\x8f\x64\x03\xbf\x5f\x63\x75\xff\xfe\x93\xeb\x0a\xab\xfb\x2f\xaa\x6f\xde\x7f\x76\x3d\xfd\xf8\xf4\xdb\x8f\x0f\x6c\x4e\x26\x39\xc3\x63\x03\xb7\xf0\xfc\xda\x9d\x1c\x9a\xdd\x57\xe8\x2a\x91\xc9\xf3\x93\x9b\x38\xad\x68\x1b\x79\x54\x81\x49\x67\xda\x09\x68\x9d\xa7\x0f\xe0\x62\x85\xe6\x66\x48\x87\x6c\x9c\xd3\xce\x07\xb9\x3b\x81\x9e\xad\xf8\x6c\xd5\xf9\xb4\x83\xf7\xb9\x1b\xbf\x6c\x88\xd9\xd9\xa3\xae\x9c\xf5\xe9\x7a\xec\x8e\xb4\x1f\xcb\x9a\xa9\x73\xa2\x7c\xf7\xc4\xcc\xb3\x61\x8a\xe2\x34\xfe\x0b\x53\xc5\xee\x89\x36\x79\xfd\x1c\x93\x3d\x34\xab\xb6\xf9\x9f\x10\xcc\xf9\x1d\x89\x14\x3f\xb8\x11\x3d\xde\x5e\xc6\x60\x06\x66\x0a\xe6\xa8\xd2\x36\x72\x8f\x9a\xff\x87\xf5\xf9\x97\x27\x5d\xe6\x47\xbb\x23\x3e\x5a\xb3\x27\x37\x96\x49\x34\xf2\x93\x5a\xfb\xdf\x0d\xe4\x29\xfa\xab\x69\x45\xf6\xa0\x32\x7e\x7b\xc8\x3e\x81\x7b\xa2\xa9\xa4\xc7\x5f\x81\x03\x68\x33\x7f\xb5\x6d\x2e\xd0\xa7\x35\x5e\x3c\x60\xa5\xd6\x6a\xfa\xa6\xda\x11\x03\x94\xaf\x0c\xdb\x80\xf2\x30\x46\xcb\x71\xf3\x9e\x8c\xc8\xdf\x01\x00\x00\xff\xff\x1f\x70\x3b\x99\x0d\x0a\x00\x00")

func installerStashAppscodeCom_stashmongodbsYamlBytes() ([]byte, error) {
	return bindataRead(
		_installerStashAppscodeCom_stashmongodbsYaml,
		"installer.stash.appscode.com_stashmongodbs.yaml",
	)
}

func installerStashAppscodeCom_stashmongodbsYaml() (*asset, error) {
	bytes, err := installerStashAppscodeCom_stashmongodbsYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "installer.stash.appscode.com_stashmongodbs.yaml", size: 2573, mode: os.FileMode(420), modTime: time.Unix(1573722179, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"installer.stash.appscode.com_stashmongodbs.v1.yaml": installerStashAppscodeCom_stashmongodbsV1Yaml,
	"installer.stash.appscode.com_stashmongodbs.yaml":    installerStashAppscodeCom_stashmongodbsYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"installer.stash.appscode.com_stashmongodbs.v1.yaml": &bintree{installerStashAppscodeCom_stashmongodbsV1Yaml, map[string]*bintree{}},
	"installer.stash.appscode.com_stashmongodbs.yaml":    &bintree{installerStashAppscodeCom_stashmongodbsYaml, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
