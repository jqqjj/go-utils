package utils

import (
	"io"
	"os"
)

type NullWriter struct {
}

func (*NullWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func MustGetFileWriter(filePath string) io.Writer {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0660)
	if err != nil {
		panic(err)
	}

	return file
}
