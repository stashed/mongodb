// Package crds Code generated by go-bindata. (@generated) DO NOT EDIT.
// sources:
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

var _installerStashAppscodeCom_stashmongodbsYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xc4\x56\x5f\x6f\xe4\x34\x10\x7f\xdf\x4f\x31\x12\x48\x07\x88\x64\x29\x27\x21\xc8\x0b\x82\x1e\x48\x27\xee\x00\x5d\x8f\x7b\xa9\x0e\x69\xd6\x9e\x64\x4d\x1d\xdb\xcc\x4c\x96\xb6\x9f\x1e\xd9\x49\xb6\xfb\xb7\x50\x04\xc2\x4f\xeb\xdf\xfc\xff\xcd\x8c\x37\x98\xdc\x3b\x62\x71\x31\x34\x80\xc9\xd1\xad\x52\xc8\x37\xa9\x6f\xbe\x94\xda\xc5\xe5\xe6\x62\x45\x8a\x17\x8b\x1b\x17\x6c\x03\x97\x83\x68\xec\xdf\x90\xc4\x81\x0d\xbd\xa0\xd6\x05\xa7\x2e\x86\x45\x4f\x8a\x16\x15\x9b\x05\x80\x61\xc2\x0c\xbe\x75\x3d\x89\x62\x9f\x1a\x08\x83\xf7\x0b\x00\x8f\x2b\xf2\x92\x75\x00\x30\xa5\x06\x44\x51\xd6\x0b\x80\x80\x3d\x4d\xb7\x3e\x86\x2e\xda\x95\xd4\x2e\x88\xa2\xf7\xc4\x75\xc1\x6b\x4c\x49\x4c\xb4\x54\x9b\xd8\x2f\x24\x91\xc9\x7e\x3a\x8e\x43\x6a\xe0\x51\xdd\xd1\xfd\x14\xd6\xa0\x52\x17\xd9\xcd\xf7\x6a\x9b\x43\xfe\x3d\xdb\x95\xeb\x58\xf2\x55\x16\xbf\xce\x49\xbd\xf8\xb6\xc0\xde\x89\xfe\x70\x24\x7a\xe5\x44\x8b\x38\xf9\x81\xd1\x1f\x14\x53\x24\xe2\x42\x37\x78\xe4\x7d\xd9\x02\x20\x31\x09\xf1\x86\x7e\x09\x37\x21\xfe\x11\xbe\x77\xe4\xad\x34\xd0\xa2\x97\x9c\x89\x98\x98\xa8\x81\x1f\x73\x11\x09\x0d\xd9\x05\xc0\x06\xbd\xb3\x85\xe5\xb1\x8c\x98\x28\x7c\xf3\xf3\xcb\x77\xcf\xaf\xcc\x9a\x7a\x1c\xc1\xec\x39\x26\x62\xdd\x56\x3b\x12\xbf\x6d\xf9\x16\x03\xb0\x24\x86\x5d\x2a\x1e\xe1\x59\x76\x35\xea\x80\xcd\x4d\x26\x01\x5d\x13\x6c\x46\x8c\x2c\x48\x09\x03\xb1\x05\x5d\x3b\x01\xa6\x52\x43\xd0\x92\xd2\x8e\x5b\xc8\x2a\x18\x20\xae\x7e\x23\xa3\x35\x5c\xe5\x3a\x59\x40\xd6\x71\xf0\x16\x4c\x0c\x1b\x62\x05\x26\x13\xbb\xe0\xee\xb7\x9e\x05\x34\x96\x90\x1e\x95\x26\x66\xe7\xe3\x82\x12\x07\xf4\x99\x84\x81\x3e\x05\x0c\x16\x7a\xbc\x03\xa6\x1c\x03\x86\xb0\xe3\xad\xa8\x48\x0d\xaf\x23\x13\xb8\xd0\xc6\x06\xd6\xaa\x49\x9a\xe5\xb2\x73\x3a\x0f\xb9\x89\x7d\x3f\x04\xa7\x77\x4b\x13\x83\xb2\x5b\x0d\x1a\x59\x96\x96\x36\xe4\x97\xe2\xba\x0a\xd9\xac\x9d\x92\xd1\x81\x69\x89\xc9\x55\x25\xf1\xa0\x65\x53\x7a\xfb\x01\x4f\x1b\x21\xcf\x76\x32\xd5\xbb\x54\x86\x9a\x5d\xe8\xb6\x70\x19\xaa\xb3\xbc\xe7\xb9\x02\x27\x80\x93\xd9\x98\xff\x03\xbd\x19\xca\xac\xbc\xf9\xee\xea\x2d\xcc\x41\x4b\x0b\xf6\x39\x2f\x6c\x3f\x98\xc9\x03\xf1\x99\x28\x17\x5a\xe2\xb1\x71\x2d\xc7\xbe\x78\xa4\x60\x53\x74\x41\xcb\xc5\x78\x47\x61\x9f\x74\x19\x56\xbd\xd3\xdc\xe9\xdf\x07\x12\xcd\xfd\xa9\xe1\x12\x43\x88\x0a\x2b\x82\x21\x59\x54\xb2\x35\xbc\x0c\x70\x89\x3d\xf9\x4b\x14\xfa\xcf\x69\xcf\x0c\x4b\x95\x29\xfd\x6b\xe2\x77\x5f\xa8\xf9\x9c\x5a\x8f\x7c\xca\x73\xb4\x87\x00\xf4\x78\xfb\x8a\x42\xa7\xeb\x06\xbe\x78\x7e\x20\x4b\xa8\x79\x24\x1b\xf8\xf5\x1a\xab\xfb\xf7\x1f\x5d\x57\x58\xdd\x7f\x56\x7d\xf5\xfe\x93\xeb\xe9\xc7\xc7\x5f\x7f\x78\x60\x73\x32\xc9\x19\x1e\x1b\xb8\x85\xe7\xd7\xee\xe4\xd0\xec\xbe\x42\x57\x89\x4c\x9e\x9f\xdc\xc4\x69\x45\xdb\xc8\xa3\x0a\x4c\x3a\xd3\x4e\x40\xeb\x3c\xfd\x0d\x2e\x56\x68\x6e\x86\x74\xc8\xc6\x39\xed\x7c\x90\xbb\x13\xe8\xd9\x8a\xf3\xc9\x53\xe5\x98\xec\xa1\x59\x55\x9c\x9d\x64\xee\x80\xa2\x7c\xda\xc1\xfb\xdc\xba\x9f\x36\xc4\xec\xec\x51\x0b\xcf\x26\xe0\x7a\xec\x8e\xb4\x1f\x2b\x91\xa9\x73\xa2\x7c\xf7\xc4\x32\xb3\x61\x8a\xe2\x34\xfe\x03\x53\xc5\xee\x5f\x63\x75\xce\xff\x84\x60\xce\xef\x48\xa4\x78\xe8\xff\x6c\x23\x7a\xbc\xbd\x8c\xc1\x0c\xcc\x14\xcc\x51\xa5\x6d\xe4\x1e\x35\xff\x69\xeb\xf3\xcf\x4f\xba\xcc\x2f\x7c\x47\x7c\xb4\x93\x4f\x6e\x2c\x93\x68\xe4\x27\xb5\xf6\x7f\x9a\xde\x53\x3e\xaa\x69\xf9\xf6\xa0\x32\xab\x7b\xc8\x3e\xdb\x7b\xa2\xa9\xfe\xc7\xdf\x97\x03\x68\x33\x7f\x0f\x6e\x2e\xd0\xa7\x35\x5e\x3c\x60\x85\x98\x6a\xfa\x5a\xdb\x11\x03\x94\xef\x17\xdb\x80\xf2\x30\x46\xcb\x71\xf3\x52\x8d\xc8\x9f\x01\x00\x00\xff\xff\x62\x1e\xa7\x1e\x67\x0a\x00\x00")

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

	info := bindataFileInfo{name: "installer.stash.appscode.com_stashmongodbs.yaml", size: 2663, mode: os.FileMode(420), modTime: time.Unix(1573722179, 0)}
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
	"installer.stash.appscode.com_stashmongodbs.yaml": installerStashAppscodeCom_stashmongodbsYaml,
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
	"installer.stash.appscode.com_stashmongodbs.yaml": &bintree{installerStashAppscodeCom_stashmongodbsYaml, map[string]*bintree{}},
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
