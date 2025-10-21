package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateTarGz archives a source folder into a gzipped tarball in memory.
func CreateTarGz(srcPath string) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Handle symlinks by following them to get the actual file
		var actualInfo os.FileInfo
		var actualPath string
		if info.Mode()&os.ModeSymlink != 0 {
			// This is a symlink, follow it
			actualPath, err = os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", path, err)
			}

			// If the symlink is relative, resolve it relative to the symlink's directory
			if !filepath.IsAbs(actualPath) {
				actualPath = filepath.Join(filepath.Dir(path), actualPath)
			}

			actualInfo, err = os.Stat(actualPath)
			if err != nil {
				return fmt.Errorf("failed to stat symlink target %s: %w", actualPath, err)
			}

			// Only follow symlinks that point to regular files
			if actualInfo.IsDir() {
				return fmt.Errorf("symlink %s points to a directory, which is not supported", path)
			}
		} else {
			actualInfo = info
			actualPath = path
		}

		// Create a tar header using the actual file info
		header, err := tar.FileInfoHeader(actualInfo, actualInfo.Name())
		if err != nil {
			return err
		}

		// Use relative paths in the archive (based on original path, not resolved path)
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If it's a regular file, write its content
		if !actualInfo.IsDir() {
			file, err := os.Open(actualPath)
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ExtractTarGz extracts a gzipped tarball from a reader to a destination directory.
func ExtractTarGz(gzipStream io.Reader, dest string) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return err
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		default:
			return fmt.Errorf("unsupported file type in tar: %c", header.Typeflag)
		}
	}
	return nil
}
