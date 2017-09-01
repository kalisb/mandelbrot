package mandelbrot

import (
	"flag"
	"image"
	"image/color"
	"sync"
	"math"
	"math/cmplx"
	"time"
  	"log"
  	"strconv"
	"io/ioutil"
	"os"
	"strings"
)

const (
	maxiteration = 1000
	e = 2.71828
)

var (
	mode    = flag.String("mode", "seq", "mode: seq, px, row, workers")
  	rect = flag.String("rect", "-2.0:2.0:-1.0:1.0", "part from the complex plane")
	quiet = flag.Bool("quiet", false, "log in quiet level")
	BSX = 32 
	BSY = 32 
	tasks = 1
	r1 float64
	r2 float64
	r3 float64
	r4 float64
)

func init() {
	flag.BoolVar(quiet, "q", false, "log in quiet level")
	flag.StringVar(rect, "r", "-2.0:2.0:-1.0:1.0", "part from the complex plane")
}

func abs(z complex128) float64 {
	return math.Hypot(real(z), imag(z))
}

type img struct {
	h, w int
	m    [][]color.RGBA
}

func (m *img) At(x, y int) color.Color { return m.m[x][y] }
func (m *img) ColorModel() color.Model { return color.RGBAModel }
func (m *img) Bounds() image.Rectangle { return image.Rect(0, 0, m.w, m.h) }

func Create(w, h, count int) image.Image {
	if (*quiet) {
		log.SetOutput(ioutil.Discard)
	}
	plane := strings.Split(*rect, ":")
	r1,_ = strconv.ParseFloat(plane[0], 64)
	r2,_ = strconv.ParseFloat(plane[1], 64)
	r3,_ = strconv.ParseFloat(plane[2], 64)
	r4,_ = strconv.ParseFloat(plane[3], 64)
	tasks = count
	c := make([][]color.RGBA, w)
	for i := range c {
		c[i] = make([]color.RGBA, h)
	}

	m := &img{h, w, c}

	switch *mode {
	case "seq":
		seqFillImg(m)
	case "px":
		oneToOneFillImg(m)
	case "row":
		onePerRowFillImg(m)
	case "workers":
		nWorkersFillImg(m)
	default:
		panic("unknown mode")
	}
	log.SetOutput(os.Stdout)
	return m
}

// sequential
func seqFillImg(m *img) {
	for i:=0; i < m.w; i++ {
		for j := 0; j < m.h; j++ {
		   fillPixel(m, i, j)
		}
	}
}

// one goroutine per pixel

func oneToOneFillImg(m *img) {
	var wg sync.WaitGroup
	wg.Add(m.h * m.w)
	for i, row := range m.m {
		for j := range row {
			go func(i, j int) {
				start := time.Now()
				log.Println("Thread-", i, ".", j, " started." )
				fillPixel(m, i, j)
				log.Println("Thread-", i, ".", j, " stopped." )
				elapsed := time.Since(start)
  				log.Println("Thread-", i, ".", j, " execution time was", elapsed)
				wg.Done()
			}(i, j)
		}
	}
	wg.Wait()
}

// one per row of pixels
func onePerRowFillImg(m *img) {
	var wg sync.WaitGroup
	for i, row := range m.m {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			log.Println("Thread-", i, " started." )
			start := time.Now()
			for j := range row {
				fillPixel(m, i, j)
			}
                        log.Println("Thread-", i, " stopped." )
			elapsed := time.Since(start)
  			log.Println("Thread-", i, " execution time was", elapsed)
		}(i)
	}
	wg.Wait()
}

func nWorkersFillImg(m *img) {
    blocks := (m.w * m.h) / (BSX * BSY)
    jobs := make(chan struct{ i, j int }, blocks)
    results := make(chan bool, blocks)


    for w := 1; w <= blocks; w++ {
        go worker(strconv.Itoa(w), m, BSX, BSY, jobs, results)
    }

    for i:=0; i < m.w; i += BSX {
	for j :=0;  j < m.h; j += BSY {
		jobs <- struct{ i, j int }{i, j}
	}
    }
	
    close(jobs)


    for a := 0; a < blocks; a++ {
        <-results
    }

}

func worker(id string, m *img, w int, h int, jobs <-chan struct{ i, j int }, results chan<- bool) {
    for t := range jobs {
	start := time.Now()
        log.Println("worker", id, "started  job - (",  t.i, ", ", t.j, ") - (", t.i + w, " ,", t.j + h, " )" )
	divide := checkBlock(t, m, w, h)
	if divide && (h > 4 || w > 4) {
		devide(id, m, w, h, t)		
	} else if (h < 4 || w < 4) {
		var limit_w = t.i + w
		if (t.i + w >= m.w) {
			limit_w = m.w - 1
		} 
		var limit_h = t.j + h
		if (t.j + h >= m.h) {
			limit_h = m.h - 1
		} 
		perPixel(m, t.i, limit_w, t.j, limit_h)
	}
        log.Println("worker", id, "finished job - (",  t.i, ", ", t.j, ") - (", t.i + w, " ,", t.j + h, " )")
	elapsed := time.Since(start)
  	log.Println("worker", id, " execution time was", elapsed)
        results <- true
    }
}

func devide(id string, m *img, w int, h int, t struct{ i, j int }) {
	h = h / 2
	w = w / 2
	subjobs := make(chan struct{ i, j int }, 4)
	subresults := make(chan bool, 4)
	for i := 0; i < 4; i++ {
		go worker(id + "-" + strconv.Itoa(i), m, w, h, subjobs, subresults)
	}

	for k:=t.i; k < t.i + w * 2; k += w {
		for l :=t.j;  l < t.j + h * 2; l += h{
			if (k < m.w && l < m.h) {
				subjobs <- struct{ i, j int }{k, l}
			}
		}
	}
	close(subjobs)
	for i := 0; i < 4; i++ {
		<-subresults
	}
}

/*
	t.i, t.j      t.i, limit_h

	limit_w, t.j   limit_w, limit_h
*/
func checkBlock(t struct{ i, j int }, m *img, w int, h int) bool{
	var limit_w = t.i + w
	if (t.i + w >= m.w) {
		limit_w = m.w - 1
	} 
	var limit_h = t.j + h
	if (t.j + h >= m.h) {
		limit_h = m.h - 1
	} 
	first := fillPixel(m, t.i, t.j)
	second := fillPixel(m, limit_w , limit_h)
	third := fillPixel(m, t.i, limit_h)
	fourth := fillPixel(m, limit_w, t.j)
	if (first == second && first == third && first == fourth) {
		for k:=t.i; k < limit_w; k++ {
			for l:=t.j; l < limit_h; l++ {
				paint(&m.m[k][l], first)
			}
		}

		return false
	} 
	return true
}


func perPixel(m *img, start_x int, end_x int, start_y int, end_y int) {
	//var wg sync.WaitGroup
	for i := start_x; i < end_x; i++ {
	//	wg.Add(1)
	//	go func(i int) {
		//	defer wg.Done()
			for j := start_y; j < end_y; j++ {
				fillPixel(m, i, j)
			}
	//	}(i)
	}
//	wg.Wait()
}

func fillPixel(m *img, i, j int) int {
	// normalized from -2 to 2
	xi := float64(i)/float64(m.w)*(math.Abs(r1) + math.Abs(r2)) + r1
	// normalized from -1 to 1
	yi := float64(j)/float64(m.h)*(math.Abs(r3) + math.Abs(r4)) + r3

	//fmt.Printf("(%g, %g) \n", xi, yi)
	
	c := complex(xi, yi)
    	calcval := int(mandel(c) * 15)

	paint(&m.m[i][j], calcval)

	return calcval;
}

func mandel(c complex128) int {
	z := complex128(0)
	for i := 0; i < maxiteration; i++ {
		if r, j := real(z), imag(z); r*r+j*j > 4 {
			return i-1
		}
		z = z*z + c * cmplx.Pow(e, -z)
	}
	return 255
}

func paint(c *color.RGBA, calcval int) {
	px_color := color.Gray{uint8(255 - calcval)}
	r, g, b, _ := px_color.RGBA()
	c.R, c.G, c.B, c.A = uint8(r), uint8(g), uint8(b), 255
}


