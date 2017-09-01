package main

import (
	"flag"
	"image/png"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"./mandelbrot"
)

var (
	size   = flag.String("size", "640x480", "size of the output image")
	tasks  = flag.Int("tasks", 2, "max number of parallel processes")
	output = flag.String("out", "zad16.png", "name of the output image file")
)

func init() {
	flag.StringVar(size, "s", "640x480", "size of the output image")
	flag.IntVar(tasks, "t", 1, "max number of parallel processes")
	flag.StringVar(output, "o", "zad16.png", "name of the output image file")
}

func main() {
	start := time.Now()

	// read the command lines params
	flag.Parse()
	runtime.GOMAXPROCS(*tasks)

	sizes := strings.Split(*size, "x")
	width, _ := strconv.Atoi(sizes[0])
	height, _ := strconv.Atoi(sizes[1])

	// open a new file
	f, err := os.Create(*output)
	if err != nil {
		log.Fatal(err)
	}
	// create the image
	img := mandelbrot.Create(width, height, *tasks)

	// and encoding it
	err = png.Encode(f, img)

	// unless you can't
	if err != nil {
		log.Fatal(err)
	}

	elapsed := time.Since(start)
	log.Printf("Execution took %s", elapsed)
}
