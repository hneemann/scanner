package scanner

import (
	"github.com/signintech/gopdf"
	"io"
)

type PdfImage struct {
	Name          string
	Width, Height int
}

func CreatePDF(w io.WriteCloser, path ...PdfImage) error {
	defer w.Close()
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	for _, p := range path {
		pdf.AddPage()

		if p.Height > (p.Width*12)/10 {
			//210 x 297
			err := pdf.Image(p.Name, 0, 0, &gopdf.Rect{W: 210 / 25.4 * 72, H: 297 / 25.4 * 72})
			if err != nil {
				return err
			}
		} else {
			h := float64((210 * p.Height) / p.Width)
			y := (297 - h) / 2

			err := pdf.Image(p.Name, 0, y/25.4*72, &gopdf.Rect{W: 210 / 25.4 * 72, H: h / 25.4 * 72})
			if err != nil {
				return err
			}
		}
	}
	_, err := pdf.WriteTo(w)
	return err
}
