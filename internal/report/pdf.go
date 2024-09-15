package report

import (
	"bufio"
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

func (r *fileReporter) GeneratePDF(userID int, start, end time.Time) ([]byte, error) {
	_, err := r.Generate(userID, start, end)
	if err != nil {
		return nil, err
	}

	p := gofpdf.New("P", "mm", "A4", "")

	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	err = p.Output(bw)
	if err != nil {
		err = fmt.Errorf("failed to write pdf to buffer: %w", err)
		return nil, err
	}

	// get the bytes from the buffer
	err = bw.Flush()
	if err != nil {
		err = fmt.Errorf("failed to flush buffer: %w", err)
	}
	return b.Bytes(), err
}
