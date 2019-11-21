package filetree

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// File stores simple file info
type File struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// Folder stores list of files and folders in the folder
type Folder struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Files   []*File   `json:"files"`
	Folders []*Folder `json:"folders"`
}

// GetFolder travel throught a folder and return a Folder
func GetFolder(root string) (*Folder, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("filetree.GetFolder: %s", err)
	}
	return getFolder(absRoot, "")
}

func getFolder(root, prefix string) (*Folder, error) {
	f, err := os.Open(root)
	if err != nil {
		return nil, fmt.Errorf("filetree.getFolder: %s", err)
	}
	folderName := filepath.Base(f.Name())
	fileInfos, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, fmt.Errorf("filetree.getFolder: %s", err)
	}

	files := []*File{}
	folders := []*Folder{}
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			name := fileInfo.Name()
			ap := path.Join(prefix, name)
			folder, err := getFolder(path.Join(root, name), ap)
			if err != nil {
				return nil, fmt.Errorf("filetree.getFolder: %s", err)
			}
			folders = append(folders, folder)
		} else {
			name := fileInfo.Name()
			ap := path.Join(prefix, name)
			files = append(files, &File{
				Name: name,
				Size: fileInfo.Size(),
				Path: ap,
			})
		}
	}
	return &Folder{
		Name:    folderName,
		Path:    prefix,
		Files:   files,
		Folders: folders,
	}, nil
}

// Print show the structure in the folder
func (f *Folder) Print() {
	f.print("")
}

func (f *Folder) print(tab string) {
	fmt.Println(tab, f.Name)
	for _, file := range f.Files {
		fmt.Println(tab, "-", file.Path, "-", file.Size, file.Name)
	}

	for _, folder := range f.Folders {
		folder.print(tab + " ")
	}
}
