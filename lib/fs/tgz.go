package fs

import (
	"path/filepath"
	"os"
	"compress/gzip"
	"fmt"
	"os/exec"
	"archive/tar"
	"io"
	"net/http"
	"github.com/jhoonb/archivex"
	"github.com/subutai-io/agent/log"
)

// Compress function creates archive file (tar.gz) of specified folder
func Compress(folder, file string) {
	archive := new(archivex.TarFile)
	archive.Create(file)
	log.Check(log.FatalLevel, "Packing file "+folder, archive.AddAll(folder, false))
	archive.Close()
}

// Decompress function extracts contents of specified archive file (tar.gz) into specified folder
// source code taken from and credits to "code.cloudfoundry.org/archiver/extractor"
func Decompress(src, dest string) error {
	srcType, err := mimeType(src)
	if err != nil {
		return err
	}

	switch srcType {
	case "application/x-gzip":
		err := extractTgz(src, dest)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s is not a tgz archive: %s", src, srcType)
	}

	return nil
}

func extractTgz(src, dest string) error {
	tarPath, err := exec.LookPath("tar")

	if err == nil {
		err := os.MkdirAll(dest, 0755)
		if err != nil {
			return err
		}

		return exec.Command(tarPath, "pzxf", src, "-C", dest).Run()
	}

	fd, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fd.Close()

	gReader, err := gzip.NewReader(fd)
	if err != nil {
		return err
	}
	defer gReader.Close()

	tarReader := tar.NewReader(gReader)
	return extractTarArchive(tarReader, dest)
}

func extractTarArchive(tarReader *tar.Reader, dest string) error {
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if hdr.Name == "." {
			continue
		}

		err = extractTarArchiveFile(hdr, dest, tarReader)
		if err != nil {
			return err
		}
	}

	return nil
}

func mimeType(src string) (string, error) {
	fd, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	data := make([]byte, 512)

	_, err = fd.Read(data)
	if err != nil {
		return "", err
	}

	return http.DetectContentType(data), nil
}

func extractTarArchiveFile(header *tar.Header, dest string, input io.Reader) error {
	filePath := filepath.Join(dest, header.Name)
	fileInfo := header.FileInfo()

	if fileInfo.IsDir() {
		return os.MkdirAll(filePath, fileInfo.Mode())
	}

	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return os.Symlink(header.Linkname, filePath)
	}

	fileCopy, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
	if err != nil {
		return err
	}

	defer fileCopy.Close()

	_, err = io.Copy(fileCopy, input)
	return err
}
