package vrctFs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CreateFile function creating file
//
// Arguments:
//
//	filePath - Path to file
//	content - content of file
//	isOptional - default option in configs / is able to merge in text files
func (v *VRCTFs) CreateFile(filePath string, content []byte, isOptional bool) error {
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	dirPath := filepath.Dir(filePath)

	err = os.MkdirAll(filepath.Join(v.virtualFSPath, dirPath), os.ModePerm)
	if err != nil {
		return err
	}

	filePrototype := FilePrototype{
		FileType: TextFile,
	}
	err = filePrototype.Read(v.virtualFSPath, filePath)
	if err != nil {
		return err
	}
	if filePrototype.FileType != TextFile {
		return errors.New("trying to create file, where it's config type")
	}

	if filePrototype.FileType != TextFile {
		return fmt.Errorf("%s cannot be created as it's already a config file", filePath)
	}

	prototypeLayer, err := filePrototype.CreateLayer(content, nil, isOptional)
	if err != nil {
		return err
	}

	err = filePrototype.AddNewLayer(prototypeLayer, false)
	return err
}

func (v *VRCTFs) ReadFile(filePath string) ([]byte, error) {
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	filePrototype := FilePrototype{}
	err = filePrototype.Read(v.virtualFSPath, filePath)
	if err != nil {
		file, err := os.ReadFile(filePath)
		if err != nil {
			return nil, os.ErrNotExist
		}

		return file, nil
	}

	if len(filePrototype.Layers) == 0 {
		return nil, os.ErrNotExist
	}

	return filePrototype.SimulateFile()
}

func (v *VRCTFs) Stat(path string) (os.FileInfo, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	splitPath := strings.Split(path, "/")
	name := splitPath[len(splitPath)-1]

	prototypePath := fmt.Sprintf("%s%s.prototype.bson", v.virtualFSPath, path)

	stat, err := os.Stat(prototypePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		filePrototype := FilePrototype{}
		if err := filePrototype.Read(v.virtualFSPath, path); err != nil {
			return nil, err
		}
		content, err := filePrototype.SimulateFile()
		if err != nil {
			return nil, err
		}

		return FileInfo{
			name:    name,
			size:    int64(len(content)),
			mode:    stat.Mode(),
			modTime: stat.ModTime(),
			isDir:   stat.IsDir(),
		}, nil
	}

	fileStat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	return FileInfo{
		name:    name,
		size:    fileStat.Size(),
		mode:    fileStat.Mode(),
		modTime: fileStat.ModTime(),
		isDir:   fileStat.IsDir(),
	}, nil
}

func (v *VRCTFs) ReadDir(path string) ([]os.DirEntry, error) {
	dirEntries := make(map[string]os.DirEntry)

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	realFsEntries, err := os.ReadDir(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		for _, entry := range realFsEntries {
			dirEntries[entry.Name()] = entry
		}
	}

	vrctEntries, err := os.ReadDir(filepath.Join(v.virtualFSPath, path))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		for _, entry := range vrctEntries {
			name := entry.Name()

			if !strings.HasSuffix(name, ".prototype.bson") && !entry.IsDir() {
				continue
			}
			if !entry.IsDir() {
				name = name[:len(name)-15]
			}

			dirEntries[name] = DirEntry{
				name:      name,
				isDir:     entry.IsDir(),
				entryType: entry.Type(),
				StatFn: func() (fs.FileInfo, error) {
					return v.Stat(path)
				},
			}
		}
	}
	res := make([]os.DirEntry, 0, len(dirEntries))

	for _, entry := range dirEntries {
		res = append(res, entry)
	}

	return res, nil
}
