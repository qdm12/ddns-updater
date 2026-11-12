package backup

import (
	"archive/zip"
	"io"
	"os"
)

func zipFiles(outputFilepath string, inputFilepaths ...string) (err error) {
	f, err := os.Create(outputFilepath)
	if err != nil {
		return err
	}
	// A zip file which failed to be written is removed, so that
	// it is not later counted as a backup to keep.
	defer func() {
		if err != nil {
			_ = os.Remove(outputFilepath)
		}
	}()
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()
	for _, filepath := range inputFilepaths {
		err = addFile(w, filepath)
		if err != nil {
			return err
		}
	}
	return nil
}

func addFile(w *zip.Writer, filepath string) error {
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
	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.
	// header.Name = filepath
	header.Method = zip.Deflate
	ioWriter, err := w.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(ioWriter, f)
	return err
}
