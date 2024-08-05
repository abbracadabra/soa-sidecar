package ioUtil

import (
	"errors"
	"fmt"
	"io"
)

type CopyError struct {
	Op  string // "read" or "write"
	Err error
}

func (e *CopyError) Error() string {
	return fmt.Sprintf("%s error: %v", e.Op, e.Err)
}

func (e *CopyError) Unwrap() error {
	return e.Err
}

func Copy(dst io.Writer, src io.Reader) (written int64, sc error, dc error) {
	buf := make([]byte, 32*1024)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return written, nil, ew
			}
			if nr != nw {
				return written, nil, io.ErrShortWrite
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				return written, nil, nil
			}
			return written, er, nil
		}
	}
}
