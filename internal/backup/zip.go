package backup

import (
	"archive/zip"
	"io"
	"os"
)

type archiveFile struct {
	path string // path of the file to archive
	name string // name of the file in the archive
}

func zipFiles(outputFilepath string, files ...archiveFile) error {
	f, err := os.Create(outputFilepath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()
	for _, file := range files {
		err = addFile(w, file.path, file.name)
		if err != nil {
			return err
		}
	}
	return nil
}

func addFile(w *zip.Writer, filepath, name string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	// Using FileInfoHeader() above only uses the basename of the file,
	// so set the name to have the same file names in all archives.
	header.Name = name
	header.Method = zip.Deflate
	ioWriter, err := w.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(ioWriter, f)
	return err
}
