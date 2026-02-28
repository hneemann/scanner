package data

import (
	"encoding/json"
	"github.com/hneemann/session/fileSys"
	"image"
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"
	"scan/scanner"
	"strconv"
	"time"
)

type DocumentType int

const (
	TypeJPEG DocumentType = iota
	TypePDF
)

type Document struct {
	Name      string
	Type      DocumentType
	Processed bool
}

type UserData struct {
	Documents []*Document

	noDownload bool
	fs         fileSys.FileSystem
}

func (d *UserData) Download() bool {
	return !d.noDownload
}

func (d *UserData) NoDownload() {
	d.noDownload = true
}

func (d *UserData) SetFileSystem(fs fileSys.FileSystem) {
	d.fs = fs
}

func (d *UserData) Save(fs fileSys.FileSystem) error {
	d.fs = fs
	w, err := d.fs.Writer("data.json")
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(d)
}

func (d *UserData) Add(data []byte) error {
	baseName := time.Now().Format("2006-01-02_15-04-05")
	name := baseName + ".jpg"
	i := 1
	for d.exists(name) {
		i++
		name = baseName + "_" + strconv.Itoa(i) + ".jpg"
	}

	w, err := d.fs.Writer(name)
	if err != nil {
		return err
	}
	defer fileSys.CloseLog(w)
	_, err = w.Write(data)
	if err != nil {
		return err
	}

	d.Documents = append(d.Documents, &Document{Name: name, Type: TypeJPEG})
	return nil
}

func (d *UserData) Create() error {
	var names []string
	for _, doc := range d.Documents {
		if doc.Type == TypeJPEG && !doc.Processed {
			names = append(names, doc.Name)
		}
	}
	log.Println("Creating PDF from", len(names), "images")
	var pdfImages []scanner.PdfImage
	for _, name := range names {
		r, err := d.fs.Reader(name)
		if err != nil {
			return err
		}
		im, _, err := image.Decode(r)
		if err != nil {
			return err
		}

		im, err = scanner.Rotate(im, false)
		if err != nil {
			return err
		}
		pdfImage, err := d.writeImage(name, im)
		if err != nil {
			return err
		}
		pdfImages = append(pdfImages, pdfImage)
	}

	pdfName := names[0] + ".pdf"
	w, err := d.fs.Writer(pdfName)
	if err != nil {
		return err
	}

	err = scanner.CreatePDF(w, pdfImages...)
	if err != nil {
		return err
	}

	d.Documents = append(d.Documents, &Document{Name: pdfName, Type: TypePDF, Processed: false})

	for _, doc := range d.Documents {
		if doc.Type == TypeJPEG {
			doc.Processed = true
		}
	}

	return nil
}

func (d *UserData) writeImage(name string, img image.Image) (scanner.PdfImage, error) {
	name = filepath.Join(os.TempDir(), "scan_"+name)
	f, err := os.Create(name)
	defer f.Close()
	err = jpeg.Encode(f, img, &jpeg.Options{Quality: 100})
	if err != nil {
		return scanner.PdfImage{}, err
	}

	return scanner.PdfImage{
		Name:   name,
		Width:  img.Bounds().Dx(),
		Height: img.Bounds().Dy(),
	}, nil
}

func Load(fs fileSys.FileSystem) (*UserData, error) {
	var data UserData
	r, err := fs.Reader("data.json")
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(r).Decode(&data)
	if err != nil {
		return nil, err
	}

	data.fs = fs
	return &data, nil
}

func (d *UserData) DeleteAll() error {
	log.Println("Deleting all documents")
	for _, doc := range d.Documents {
		err := d.fs.Delete(doc.Name)
		if err != nil {
			log.Println("Error deleting document", doc.Name, err)
		}
	}
	d.Documents = d.Documents[0:0]
	return nil
}

func (d *UserData) Delete(index int) error {
	if index < 0 || index >= len(d.Documents) {
		return nil
	}
	log.Println("Deleting document", index, d.Documents[index].Name)
	doc := d.Documents[index]
	err := d.fs.Delete(doc.Name)
	if err != nil {
		return err
	}
	d.Documents = append(d.Documents[:index], d.Documents[index+1:]...)
	return nil
}

func (d *UserData) Reader(name string) (io.ReadCloser, error) {
	return d.fs.Reader(name)
}

func (d *UserData) exists(name string) bool {
	for _, doc := range d.Documents {
		if doc.Name == name {
			return true
		}
	}
	return false
}
