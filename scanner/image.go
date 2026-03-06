package scanner

import (
	"errors"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"math"
	"sort"
)

type Vector struct {
	X, Y int
}

func (v Vector) add(dif Vector) Vector {
	return Vector{
		X: v.X + dif.X,
		Y: v.Y + dif.Y,
	}
}
func (v Vector) sub(dif Vector) Vector {
	return Vector{
		X: v.X - dif.X,
		Y: v.Y - dif.Y,
	}
}
func (v Vector) abs() float64 {
	return math.Sqrt(float64(v.X*v.X + v.Y*v.Y))
}

func (v Vector) cosAngle(o Vector) float64 {
	vl := v.abs()
	ol := o.abs()
	return float64(v.X*o.X+v.Y*o.Y) / (vl * ol)
}

func (v Vector) sMul(f int) Vector {
	return Vector{
		X: v.X * f,
		Y: v.Y * f,
	}
}

func (v Vector) sDiv(d int) Vector {
	return Vector{
		X: v.X / d,
		Y: v.Y / d,
	}
}

func calcEdgeList(img image.Image, step int) ([]Vector, []Vector) {
	xMin := img.Bounds().Min.X
	yMin := img.Bounds().Min.Y
	xMax := img.Bounds().Max.X
	yMax := img.Bounds().Max.Y

	var upEdgeList []Vector
	var downEdgeList []Vector

	yStart := step
	scanLine := make([]float64, xMax-xMin+1)
	for {
		yInitial := yMin + yStart
		xInitial := xMin
		if yInitial > yMax {
			dy := yInitial - yMax
			xInitial += dy
			yInitial -= dy
			if xInitial > xMax {
				break
			}
		}
		maxGray := 0.0
		x := xInitial
		y := yInitial
		n := 0
		for x <= xMax && y >= yMin {
			r, g, b, _ := img.At(x, y).RGBA()
			gray := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 256
			scanLine[x-xInitial] = gray
			if gray > maxGray {
				maxGray = gray
			}
			n++
			x++
			y--
		}
		stateMachine(scanLine, n, 100, &upEdgeList, &downEdgeList, xInitial, yInitial)

		yStart += step
	}
	return upEdgeList, downEdgeList
}

func stateMachine(scanLine []float64, n int, mean float64, upEdge, downEdge *[]Vector, xInitial int, yInitial int) {
	const threshold = 20
	dark := true
	lastSameCounter := 0
	sameCounter := 0
	for i := 0; i < n; i++ {
		if dark {
			if scanLine[i] < mean {
				sameCounter++
				if sameCounter == threshold && lastSameCounter >= threshold {
					pos := i - sameCounter
					*upEdge = append(*upEdge, Vector{X: xInitial + pos, Y: yInitial - pos})
				}
			} else {
				dark = false
				lastSameCounter = sameCounter
				sameCounter = 0
			}
		} else {
			if scanLine[i] > mean {
				sameCounter++
				if sameCounter == threshold && lastSameCounter >= threshold {
					pos := i - sameCounter
					*downEdge = append(*downEdge, Vector{X: xInitial + pos, Y: yInitial - pos})
				}
			} else {
				dark = true
				lastSameCounter = sameCounter
				sameCounter = 0
			}
		}
	}
}

type transform struct {
	tl     Vector
	tr     Vector
	bl     Vector
	br     Vector
	width  int
	height int
}

func (t transform) transform(x int, y int) (int, int) {
	xt := t.tr.sub(t.tl).sMul(x).sDiv(t.width).add(t.tl)
	xb := t.br.sub(t.bl).sMul(x).sDiv(t.width).add(t.bl)

	p := xb.sub(xt).sMul(y).sDiv(t.height).add(xt)
	return p.X, p.Y
}

func Rotate(img image.Image, debug bool) (image.Image, error) {

	edges1, edges2 := calcEdgeList(img, 40)

	l1, l2, err := toLines(edges1)
	if err != nil {
		return nil, err
	}
	l4, l3, err := toLines(edges2)
	if err != nil {
		return nil, err
	}

	var debugImage *image.RGBA
	if debug {
		debugImage = image.NewRGBA(image.Rect(0, 0, img.Bounds().Max.X, img.Bounds().Max.Y))
		draw.Draw(debugImage, img.Bounds(), img, image.Point{}, draw.Src)

		for _, v := range edges1 {
			point(debugImage, v, image.Black)
		}
		for _, v := range edges2 {
			point(debugImage, v, image.Black)
		}
	}

	l1Col := color.RGBA{255, 0, 0, 255}
	l2Col := color.RGBA{0, 255, 0, 255}
	l3Col := color.RGBA{255, 0, 255, 255}
	l4Col := color.RGBA{0, 0, 255, 255}

	if debug {
		for _, v := range l1.points {
			point(debugImage, v, l1Col)
		}
		for _, v := range l2.points {
			point(debugImage, v, l2Col)
		}
		for _, v := range l3.points {
			point(debugImage, v, l3Col)
		}
		for _, v := range l4.points {
			point(debugImage, v, l4Col)
		}
		drawLine(debugImage, l1.l, l1Col)
		drawLine(debugImage, l2.l, l2Col)
		drawLine(debugImage, l3.l, l3Col)
		drawLine(debugImage, l4.l, l4Col)
	}

	tl := l1.l.intersect(l3.l)
	tr := l1.l.intersect(l2.l)
	bl := l3.l.intersect(l4.l)
	br := l2.l.intersect(l4.l)

	if debug {
		point(debugImage, tl, color.RGBA{255, 255, 0, 255})
		point(debugImage, tr, color.RGBA{255, 255, 0, 255})
		point(debugImage, bl, color.RGBA{255, 255, 0, 255})
		point(debugImage, br, color.RGBA{255, 255, 0, 255})

		return debugImage, nil
	} else {
		width := int(math.Max(tl.sub(tr).abs(), bl.sub(br).abs()))
		height := int(math.Max(tl.sub(bl).abs(), br.sub(br).abs()))

		newImage := image.NewRGBA(image.Rect(0, 0, width, height))
		rect := img.Bounds()

		trans := transform{tl: tl, tr: tr, bl: bl, br: br, width: width, height: height}
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				xx, yy := trans.transform(x, y)

				var c color.Color
				if !(image.Point{xx, yy}.In(rect)) {
					c = color.White
				} else {
					c = img.At(xx, yy)
				}

				newImage.Set(x, y, c)
			}
		}
		return newImage, nil
	}

}

type segment struct {
	angle  float64
	points []Vector
	l      line
}

func (s *segment) addPoint(p Vector) {
	for _, i := range s.points {
		if i.X == p.X && i.Y == p.Y {
			return // already exists
		}
	}
	s.points = append(s.points, p)
}

func newSegment(p ...Vector) segment {
	li := calcRegression(p)
	return segment{
		angle:  li.angle(),
		points: p,
		l:      li,
	}
}

type segments []segment

func (s *segments) addPointTo(i int, p ...Vector) {
	se := &(*s)[i]
	for _, po := range p {
		se.addPoint(po)
	}
	se.l = calcRegression(se.points)
	se.angle = se.l.angle()
}

func toLines(points []Vector) (segment, segment, error) {
	var segm segments

	for p := 1; p < len(points); p++ {
		d := points[p-1].sub(points[p])
		angle := normAngle(math.Atan2(float64(d.Y), float64(d.X)))
		found := false
		for i := 0; i < len(segm); i++ {
			dist := segm[i].l.dist(points[p-1])
			if math.Abs(segm[i].angle-angle) < 0.1 && dist < 40 {
				segm.addPointTo(i, points[p-1], points[p])
				found = true
			}
		}
		if !found {
			segm = append(segm, newSegment(points[p-1], points[p]))
		}
	}
	sort.Slice(segm, func(i, j int) bool {
		return len(segm[i].points) > len(segm[j].points)
	})

	if len(segm) < 2 {
		return segment{}, segment{}, errors.New("could not find a page: not enough edges found")
	}

	var s0 *segment
	var s1 *segment
	for _, s := range segm {
		if s0 == nil && !s.l.yIsVar {
			s0 = &s
		}
		if s1 == nil && s.l.yIsVar {
			s1 = &s
		}
	}
	if s0 == nil || s1 == nil {
		return segment{}, segment{}, errors.New("could not a find page: no perpendicular edges")
	}

	return *s0, *s1, nil
}

func normAngle(angle float64) float64 {
	const max = 3 * math.Pi / 4
	const min = -math.Pi / 4

	if angle > max {
		angle -= math.Pi
		if angle > max {
			angle -= math.Pi
		}
	}
	if angle <= min {
		angle += math.Pi
	}
	return angle
}

func point(newImage *image.RGBA, v Vector, col color.Color) {
	const rad = 8
	for x := -rad; x < rad; x++ {
		for y := -rad; y < rad; y++ {
			newImage.Set(v.X+x, v.Y+y, col)
		}
	}
}

func drawLine(newImage *image.RGBA, l line, color color.Color) {
	if l.yIsVar {
		yMax := newImage.Bounds().Max.Y
		for y := 0; y < yMax; y++ {
			newImage.Set(l.x(y), y, color)
		}
	} else {
		xMax := newImage.Bounds().Max.X
		for x := 0; x < xMax; x++ {
			newImage.Set(x, l.y(x), color)
		}
	}
}

type line struct {
	a      float64
	b      float64
	yIsVar bool
}

func (l line) intersect(b line) Vector {
	var x, y float64
	if l.yIsVar {
		if b.yIsVar {
			if l.a == b.a {
				return Vector{X: -1, Y: -1}
			}
			y = (b.b - l.b) / (l.a - b.a)
			x = l.a*y + l.b
		} else {
			x = (l.a*b.b + l.b) / (1 - l.a*b.a)
			y = b.a*x + b.b
		}
	} else {
		if b.yIsVar {
			x = (b.a*l.b + b.b) / (1 - l.a*b.a)
			y = l.a*x + l.b
		} else {
			if l.a == b.a {
				return Vector{X: -1, Y: -1}
			}
			x = (b.b - l.b) / (l.a - b.a)
			y = l.a*x + l.b
		}
	}
	return Vector{X: int(x), Y: int(y)}
}

func (l line) y(x int) int {
	if l.yIsVar {
		return int((float64(x) - l.b) / l.a)
	}
	return int(l.a*float64(x) + l.b)
}

func (l line) x(y int) int {
	if l.yIsVar {
		return int(l.a*float64(y) + l.b)
	}
	return int((float64(y) - l.b) / l.a)
}

func (l line) angle() float64 {
	angle := math.Atan(l.a)
	if l.yIsVar {
		angle = math.Pi/2 - angle
	}
	return normAngle(angle)
}

func (l line) dist(p Vector) int {
	if l.yIsVar {
		return int(math.Abs(float64(p.X - l.x(p.Y)))) // x = a*y + b
	}
	return int(math.Abs(float64(p.Y - l.y(p.X)))) // y = a*x + b
}

func calcRegression(data []Vector) line {
	var sxi float64
	var syi float64
	var sxi2 float64
	var syi2 float64
	var sxiyi float64
	n := 0
	var xMin, xMax, yMin, yMax int
	for i, p := range data {
		if i == 0 {
			xMin = p.X
			xMax = p.X
			yMin = p.Y
			yMax = p.Y
		} else {
			if p.X < xMin {
				xMin = p.X
			}
			if p.X > xMax {
				xMax = p.X
			}
			if p.Y < yMin {
				yMin = p.Y
			}
			if p.Y > yMax {
				yMax = p.Y
			}
		}
		sxi += float64(p.X)
		syi += float64(p.Y)
		sxi2 += float64(p.X) * float64(p.X)
		syi2 += float64(p.Y) * float64(p.Y)
		sxiyi += float64(p.X) * float64(p.Y)
		n++
	}

	dx := xMax - xMin
	dy := yMax - yMin

	if dx > dy {
		div := sxi2 - sxi*sxi/float64(n)
		a := (sxiyi - sxi*syi/float64(n)) / div
		b := (syi - a*sxi) / float64(n)
		return line{
			a:      a,
			b:      b,
			yIsVar: false,
		}
	} else {
		div := syi2 - syi*syi/float64(n)
		a := (sxiyi - sxi*syi/float64(n)) / div
		b := (sxi - a*syi) / float64(n)
		return line{
			a:      a,
			b:      b,
			yIsVar: true,
		}
	}
}
