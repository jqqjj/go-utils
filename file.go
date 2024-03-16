package utils

import (
	"io"
	"os"
	"path"
	"path/filepath"
)

func FileExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func FileSize(filename string) (int64, error) {
	f, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}
	return f.Size(), nil
}

func FileIsRegular(filename string) bool {
	f, err := os.Stat(filename)
	return err == nil && !f.IsDir()
}

func FileIsDir(filename string) bool {
	f, err := os.Stat(filename)
	return err == nil && f.IsDir()
}

func GetSymlinkRealPath(file string) (string, bool) {
	f, err := os.Lstat(file)
	if err != nil {
		return "", false
	}
	if f.Mode()&os.ModeSymlink == 0 {
		return "", false

	}
	realPath, err := os.Readlink(file)
	if err != nil {
		return "", false
	}
	return realPath, true
}

func GetFiles(pathname string, recursive bool) ([]string, error) {
	var (
		err       error
		fileInfos []os.DirEntry

		files, subFiles []string
	)

	if fileInfos, err = os.ReadDir(pathname); err != nil {
		return nil, err
	}

	// 所有文件/文件夹
	for _, fi := range fileInfos {
		// 是文件夹则递归进入获取;是文件，则压入数组
		if fi.IsDir() && recursive {
			subDir := fi.Name()
			if subFiles, err = GetFiles(path.Join(pathname, subDir), recursive); err != nil {
				return nil, err
			}
			for _, v := range subFiles {
				files = append(files, path.Join(subDir, v))
			}
		} else {
			files = append(files, fi.Name())
		}
	}

	return files, nil
}

func MoveFile(sourcePath, destPath string) error {
	var (
		err error

		srcFile, dstFile *os.File
	)

	defer func() {
		if err == nil { //移动文件无错误删除原路径文件
			_ = os.Remove(sourcePath)
		}
	}()

	if srcFile, err = os.OpenFile(sourcePath, os.O_RDONLY, 0660); err != nil {
		return err
	}
	defer srcFile.Close()

	if dstFile, err = os.Create(destPath); err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return nil
}

func DirectoryDelete(dirPath string) error {
	// 递归获取目录下的所有文件和子目录
	files, err := filepath.Glob(filepath.Join(dirPath, "*"))
	if err != nil {
		return err
	}

	// 删除文件
	for _, file := range files {
		err = os.RemoveAll(file)
		if err != nil {
			return err
		}
	}

	// 删除目录本身
	err = os.RemoveAll(dirPath)
	if err != nil {
		return err
	}

	return nil
}
