package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractZipEntry copies the first file whose base name matches baseName from
// the zip archive to dest.
func extractZipEntry(archivePath, baseName, dest string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || filepath.Base(f.Name) != baseName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return writeFile(dest, rc)
	}
	return fmt.Errorf("%q not found in archive", baseName)
}

// extractTarEntry copies the first file whose base name matches baseName from
// the gzip-compressed tar archive to dest.
func extractTarEntry(archivePath, baseName, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg || filepath.Base(h.Name) != baseName {
			continue
		}
		return writeFile(dest, tr)
	}
	return fmt.Errorf("%q not found in archive", baseName)
}

// unzipAll extracts an entire zip archive into destDir, preserving
// directories, file modes, and symlinks.
func unzipAll(archivePath, destDir string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		target := filepath.Join(destDir, f.Name)
		// Guard against path traversal ("zip slip").
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe path in archive: %q", f.Name)
		}
		info := f.FileInfo()
		switch {
		case info.IsDir():
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case info.Mode()&os.ModeSymlink != 0:
			if err := extractSymlink(f, target); err != nil {
				return err
			}
		default:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			err = writeFileMode(target, rc, info.Mode())
			rc.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func extractSymlink(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	linkBytes, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	os.Remove(target)
	return os.Symlink(string(linkBytes), target)
}

func writeFile(dest string, r io.Reader) error {
	return writeFileMode(dest, r, 0o644)
}

func writeFileMode(dest string, r io.Reader, mode os.FileMode) error {
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, r)
	return err
}
