package scanner

import (
	"github.com/jung-kurt/gofpdf"
	"io"
)

func CreatePDF(w io.WriteCloser, path ...string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	for _, p := range path {
		pdf.AddPage()
		//210 x 297
		pdf.Image(p, 0, 0, 210, 297, false, "", 0, "")
	}
	return pdf.OutputAndClose(w)
}
