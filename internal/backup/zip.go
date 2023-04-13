package backup

import (
	"archive/zip"
	"io"
	"os"
)

var _ FileZiper = (*Ziper)(nil)

type FileZiper interface {
	ZipFiles(outputFilepath string, inputFilepaths ...string) error
}

type Ziper struct {
	createFile func(name string) (*os.File, error)
	openFile   func(name string) (*os.File, error)
	ioCopy     func(dst io.Writer, src io.Reader) (written int64, err error)
}

func NewZiper() *Ziper {
	return &Ziper{
		createFile: os.Create,
		openFile:   os.Open,
		ioCopy:     io.Copy,
	}
}

func (z *Ziper) ZipFiles(outputFilepath string, inputFilepaths ...string) error {
	f, err := z.createFile(outputFilepath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()
	for _, filepath := range inputFilepaths {
		err = z.addFile(w, filepath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (z *Ziper) addFile(w *zip.Writer, filepath string) error {
	f, err := z.openFile(filepath)
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
	_, err = z.ioCopy(ioWriter, f)
	return err
}
