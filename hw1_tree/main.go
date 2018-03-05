package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort" // sorter
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(in io.Writer, path string, printFiles bool) error {
	dir, err := filepath.Abs(path) // get the absolute path
	if err != nil {
		return err
	}
	tree, err := walkFiles(dir)	// construct the file tree
	if err != nil {
		return err
	}
	printTree(in, tree, printFiles)

	return nil // no errors
}

// ---

type dirFile struct {
	name string	// file / directory name
	size int64	// file size
	isDir bool	// is the file a directory
	// ---
	subDirFiles []*dirFile	// recursion
}

// traverse file dir
func walkFiles(path string) (*dirFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() // close the file descriptor (not sure if needed)
	fileInfo, err := f.Stat()	// info about the file (name, size)
	if err != nil {
		return nil, err
	}
	dirFile := &dirFile {
		name: fileInfo.Name(),
		size: fileInfo.Size(),
		isDir: fileInfo.IsDir(),	// trailing comma is required (do not remove)
	}

	if fileInfo.IsDir() {
		// get sub-file names -> paths
		subFileNames, err := f.Readdirnames(-1)
		if err != nil {
			return nil, err
		}
		// fmt.Println(subFileNames)
		for _, subFileName := range subFileNames {
			// construct the file path
			filePath := filepath.Join(path, subFileName)
			// recursive call
			subFiles, err := walkFiles(filePath)
			if err != nil {
				return nil, err
			}
			dirFile.subDirFiles = append(dirFile.subDirFiles, subFiles)
		}
	}

	return dirFile, nil	// no errors
}

// facade
func printTree(out io.Writer, root *dirFile, printFiles bool) {
	printTreeFile(out, root, printFiles, "")
}

// recursive; should be error-free
func printTreeFile(out io.Writer, root *dirFile, printFiles bool, dirPrefix string) {
	files := root.subDirFiles	// alias
	// sort lexicographically (i.e alphabetically)
	sort.Slice(files, func(i, j int) bool {
		return files[i].name < files[j].name
	})

	var lastIdx int
	for idx, entry := range files {
 		if entry.isDir || printFiles {
			lastIdx = idx
		}
 	}

	// loop through the list of files
	for idx, entry := range files {
		// prefix management
		var prefix, nextLevelPrefix string

		// skip the 'file' if not directory and printFiles = false
		if !entry.isDir && !printFiles {
			continue
		}

		// directory symbol
		if idx == lastIdx {
			prefix = dirPrefix + "└───"
			nextLevelPrefix = dirPrefix + "\t"
		} else {
			prefix = dirPrefix + "├───"
			nextLevelPrefix = dirPrefix + "│\t"
		}

		// print the entry information - name and size (only for files)
		var size string // empty string
		if !entry.isDir {	// is file?
			if entry.size > 0 {
				size = fmt.Sprintf(" (%db)", entry.size)	// notice space
			} else {
				size = " (empty)" // notice space
			}
		}
		// to remove the extra space in-between
		line := prefix + entry.name + size
		fmt.Fprintln(out, line)	// Println is NOT to be used

		// recursive call
		if entry.isDir {
			printTreeFile(out, entry, printFiles, nextLevelPrefix)
		}
	}
}
