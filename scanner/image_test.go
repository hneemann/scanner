package scanner

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func getImageFromFilePath(t *testing.T, name string) image.Image {
	f, err := os.Open("testdata/" + name)
	assert.NoError(t, err, "Failed to open image file: %s", name)
	defer f.Close()
	im, _, err := image.Decode(f)
	assert.NoError(t, err, "Failed to decode image file: %s", name)
	return im
}

func TestFirst(t *testing.T) {
	got := getImageFromFilePath(t, "i1.jpg")
	rotate, err := Rotate(got, false)
	assert.NoError(t, err, "Failed to rotate image: i1.jpg")
	path1, err := writeImage("i1", rotate)
	assert.NoError(t, err)

	got = getImageFromFilePath(t, "i2.jpg")
	rotate, err = Rotate(got, false)
	assert.NoError(t, err, "Failed to rotate image: i2.jpg")
	path2, err := writeImage("i2", rotate)
	assert.NoError(t, err)

	got = getImageFromFilePath(t, "i3.jpg")
	rotate, err = Rotate(got, false)
	assert.NoError(t, err, "Failed to rotate image: i3.jpg")
	path3, err := writeImage("i3", rotate)
	assert.NoError(t, err)

	err = CreatePDF("/home/hneemann/temp/scan/z.pdf", path1, path2, path3)
	assert.NoError(t, err)
}

func TestDebug(t *testing.T) {
	folder := "/home/hneemann/temp/scan/admin"
	f, err := os.Open(filepath.Join(folder, "2025-08-21_08-22-33.jpg"))
	assert.NoError(t, err)
	defer f.Close()
	im, _, err := image.Decode(f)
	rotate, err := Rotate(im, true)
	assert.NoError(t, err, "Failed to rotate image")
	_, err = writeImage("z", rotate)
	assert.NoError(t, err)
}

func writeImage(name string, img image.Image) (string, error) {
	if img == nil {
		return "", fmt.Errorf("image is nil")
	}

	path := "/home/hneemann/temp/scan/" + name + ".jpg"
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return path, jpeg.Encode(f, img, &jpeg.Options{Quality: 100})
}

func TestIntegration(t *testing.T) {
	folder := "/home/hneemann/temp/scan/admin"
	f, err := os.Open(folder)
	assert.NoError(t, err, "Failed to open folder: %s", folder)
	list, err := f.ReadDir(-1)
	assert.NoError(t, err, "Failed to read directory: %s", folder)
	for _, file := range list {
		if file.IsDir() {
			continue
		}
		if strings.HasSuffix(file.Name(), ".jpg") {
			fmt.Println("Processing file:", file.Name())
			f, err := os.Open(filepath.Join(folder, file.Name()))
			assert.NoError(t, err, "Failed to open image file: %s", file.Name())
			defer f.Close()
			im, _, err := image.Decode(f)
			assert.NoError(t, err, "Failed to decode image file: %s", file.Name())
			rotate, err := Rotate(im, false)
			assert.NoError(t, err, "Failed to rotate image: %s", file.Name())
			_, err = writeImage(file.Name(), rotate)
			assert.NoError(t, err, "Failed to write rotated image: %s", file.Name())
		}
	}
}

func Test_calcRegression(t *testing.T) {
	tests := []struct {
		name string
		data []Vector
	}{
		{name: "simple", data: []Vector{{X: 0, Y: 0}, {X: 200, Y: 100}, {X: 400, Y: 200}, {X: 600, Y: 300}}},
		{name: "up", data: []Vector{{X: 0, Y: 0}, {X: 100, Y: 200}, {X: 200, Y: 400}, {X: 300, Y: 600}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			li := calcRegression(tt.data)
			for _, p := range tt.data {
				x := li.x(p.Y)
				y := li.y(p.X)
				assert.InDelta(t, p.X, x, 1, "X value does not match for point %v", p)
				assert.InDelta(t, p.Y, y, 1, "Y value does not match for point %v", p)

				d := li.dist(p)
				assert.InDelta(t, 0, d, 1, "Distance does not match for point %v", p)

				if li.yIsVar {
					d = li.dist(Vector{p.X + 10, p.Y})
					assert.InDelta(t, 10, d, 1, "Distance does not match for point %v", p)
					d = li.dist(Vector{p.X, p.Y + 10})
					assert.InDelta(t, 5, d, 1, "Distance does not match for point %v", p)
				} else {
					d = li.dist(Vector{p.X, p.Y + 10})
					assert.InDelta(t, 10, d, 1, "Distance does not match for point %v", p)
					d = li.dist(Vector{p.X + 10, p.Y})
					assert.InDelta(t, 5, d, 1, "Distance does not match for point %v", p)
				}

			}
		})
	}
}

func Test_line_intersect(t *testing.T) {
	tests := []struct {
		name string
		a, b line
		want Vector
	}{
		{
			name: "hori",
			a:    calcRegression([]Vector{{X: 0, Y: 0}, {X: 100, Y: 0}}),
			b:    calcRegression([]Vector{{X: 0, Y: -50}, {X: 100, Y: 0}}),
			want: Vector{X: 100, Y: 0},
		},
		{
			name: "hori2",
			a:    calcRegression([]Vector{{X: 0, Y: 50}, {X: 100, Y: 0}}),
			b:    calcRegression([]Vector{{X: 0, Y: -50}, {X: 100, Y: 0}}),
			want: Vector{X: 100, Y: 0},
		},
		{
			name: "vert",
			a:    calcRegression([]Vector{{X: 0, Y: 0}, {X: 0, Y: 100}}),
			b:    calcRegression([]Vector{{X: -50, Y: 0}, {X: 0, Y: 100}}),
			want: Vector{X: 0, Y: 100},
		},
		{
			name: "vert2",
			a:    calcRegression([]Vector{{X: 50, Y: 0}, {X: 0, Y: 100}}),
			b:    calcRegression([]Vector{{X: -50, Y: 0}, {X: 0, Y: 100}}),
			want: Vector{X: 0, Y: 100},
		},
		{
			name: "mix1",
			a:    calcRegression([]Vector{{X: 0, Y: 0}, {X: 100, Y: 0}}),
			b:    calcRegression([]Vector{{X: 100, Y: 0}, {X: 100, Y: 100}}),
			want: Vector{X: 100, Y: 0},
		},
		{
			name: "mix1b",
			a:    calcRegression([]Vector{{X: 0, Y: 10}, {X: 100, Y: 0}}),
			b:    calcRegression([]Vector{{X: 100, Y: 0}, {X: 100, Y: 90}}),
			want: Vector{X: 100, Y: 0},
		},
		{
			name: "mix2",
			a:    calcRegression([]Vector{{X: 100, Y: 0}, {X: 100, Y: 100}}),
			b:    calcRegression([]Vector{{X: 0, Y: 0}, {X: 100, Y: 0}}),
			want: Vector{X: 100, Y: 0},
		},
		{
			name: "mix2b",
			a:    calcRegression([]Vector{{X: 100, Y: 0}, {X: 90, Y: 100}}),
			b:    calcRegression([]Vector{{X: 0, Y: 20}, {X: 100, Y: 0}}),
			want: Vector{X: 100, Y: 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, tt.want, tt.a.intersect(tt.b))
			assert.EqualValues(t, tt.want, tt.b.intersect(tt.a))
		})
	}
}

func Test_calcRegression1(t *testing.T) {
	tests := []struct {
		name string
		data []Vector
		want line
	}{
		{
			name: "test",
			data: []Vector{{2048, 592}, {2048, 632}, {2047, 673}, {2047, 713}, {2046, 754}, {2046, 794}, {2046, 834}, {2045, 875}, {2046, 914}, {2045, 995}, {2044, 1036}, {2043, 1077}, {2043, 1117}, {2042, 1158}, {2041, 1199}, {2041, 1239}, {2041, 1279}, {2040, 1320}, {2040, 1360}, {2040, 1400}, {2040, 1440}, {2041, 1479}, {2042, 1518}, {2042, 1558}, {2042, 1598}, {2042, 1638}, {2042, 1678}, {2043, 1717}, {2043, 1757}, {2043, 1797}, {2044, 1836}, {2044, 1876}, {2045, 1915}, {2046, 1954}, {2046, 1994}, {2047, 2033}, {2048, 2072}, {2049, 2111}},
			want: line{a: -0.000734554, b: 2044.9, yIsVar: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regression := calcRegression(tt.data)
			assert.InDelta(t, tt.want.a, regression.a, 1e-5, "calcRegression(%v)", tt.data)
			assert.InDelta(t, tt.want.b, regression.b, 1e-2, "calcRegression(%v)", tt.data)
		})
	}
}

func Test_angle(t *testing.T) {
	for i := 0; i < 31; i++ {
		p0 := Vector{X: 10, Y: 10}
		angle := float64(i) * 2 * math.Pi / 32
		p1 := Vector{X: int(float64(p0.X) + 1000*math.Cos(angle)), Y: int(float64(p0.Y) + 1000*math.Sin(angle))}

		a1 := normAngle(angle)
		atan2 := math.Atan2(float64(p1.Y-p0.Y), float64(p1.X-p0.X))
		a2 := normAngle(atan2)

		assert.InDelta(t, a1, a2, 0.001, "Angle does not match for angle %f", angle)

		calc := calcRegression([]Vector{p0, p1})
		a3 := calc.angle()
		assert.InDelta(t, a1, a3, 0.001, "Angle does not match for angle %f", angle)

	}
}
