package scanner

import (
	"github.com/jung-kurt/gofpdf"
	"io"
)

type PdfImage struct {
	Name          string
	Width, Height int
}

func CreatePDF(w io.WriteCloser, path ...PdfImage) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	for _, p := range path {
		pdf.AddPage()

		if p.Height > (p.Width*12)/10 {
			//210 x 297
			pdf.Image(p.Name, 0, 0, 210, 297, false, "", 0, "")
		} else {
			h := float64((210 * p.Height) / p.Width)
			y := (297 - h) / 2

			pdf.Image(p.Name, 0, y, 210, h, false, "", 0, "")
		}
	}
	return pdf.OutputAndClose(w)
}
