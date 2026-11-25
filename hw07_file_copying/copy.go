package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrUnsupportedFile       = errors.New("unsupported file")
	ErrOffsetExceedsFileSize = errors.New("offset exceeds file size")
	ErrInvalidOffset         = errors.New("invalid offset")
	ErrInvalidLimit          = errors.New("invalid limit")
	ErrSameFile              = errors.New("source and destination are the same file")
)

const copyBufferSize = 4 * 1024

var progressOutput io.Writer = os.Stderr

func Copy(fromPath, toPath string, offset, limit int64) error {
	if offset < 0 {
		return ErrInvalidOffset
	}
	if limit < 0 {
		return ErrInvalidLimit
	}

	src, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return ErrUnsupportedFile
	}

	if toInfo, statErr := os.Stat(toPath); statErr == nil && os.SameFile(info, toInfo) {
		return ErrSameFile
	}

	if offset > info.Size() {
		return ErrOffsetExceedsFileSize
	}
	if _, err := src.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	toCopy := info.Size() - offset
	if limit > 0 && limit < toCopy {
		toCopy = limit
	}

	dst, err := os.Create(toPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	progress := newProgressWriter(dst, progressOutput, toCopy)
	_, copyErr := io.CopyBuffer(progress, io.LimitReader(src, toCopy), make([]byte, copyBufferSize))
	progress.finish()

	return copyErr
}

type progressWriter struct {
	writer      io.Writer
	output      io.Writer
	total       int64
	copied      int64
	lastPercent int
}

func newProgressWriter(writer, output io.Writer, total int64) *progressWriter {
	progress := &progressWriter{
		writer:      writer,
		output:      output,
		total:       total,
		lastPercent: -1,
	}
	progress.show()

	return progress
}

func (p *progressWriter) Write(data []byte) (int, error) {
	written, err := p.writer.Write(data)
	p.copied += int64(written)
	p.show()

	return written, err
}

func (p *progressWriter) show() {
	if p.total <= 0 {
		return
	}

	percent := int(p.copied * 100 / p.total)
	if percent == p.lastPercent {
		return
	}
	p.lastPercent = percent

	_, _ = fmt.Fprintf(p.output, "\r%d%%", percent)
}

func (p *progressWriter) finish() {
	if p.total <= 0 {
		return
	}

	p.show()
	_, _ = fmt.Fprintln(p.output)
}
