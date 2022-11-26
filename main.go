package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gen2brain/go-fitz"
	"golang.org/x/sync/errgroup"
)

var (
	source      = flag.String("s", "./", "source file path; either a directory containing PDF files or a single PDF file")
	concurrency = flag.Int("c", 4, "max number of PDFs that will be processed concurrently")
)

type pdfStats struct {
	name                  string
	whitePixels           int64
	nonWhitePixels        int64
	percentageWhitePixels float64
}

func main() {
	flag.Parse()

	ctx := context.Background()
	g, _ := errgroup.WithContext(ctx)
	pdfFilePaths := make(chan string)
	var allStats []*pdfStats
	var mu sync.Mutex

	g.Go(func() error {
		defer close(pdfFilePaths)

		log.Printf("Reading PDFs from source %s", *source)

		err := forEachPdf(*source, func(path string) {
			log.Printf("Queueing PDF for processing: %s", path)
			pdfFilePaths <- path
		})

		return err
	})

	tempDir, err := os.MkdirTemp(
		os.TempDir(), "pdf-analyzer")
	if err != nil {
		exitWithError(fmt.Errorf("unable to create temp directory for converting PDFs to images: %w", err))
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			exitWithError(fmt.Errorf("error removing temp directory used to convert PDFs to images: %w", err))
		}
	}()

	for i := 0; i < *concurrency; i++ {
		g.Go(func() error {
			for path := range pdfFilePaths {
				localStats, err := processPdf(tempDir, path)
				if err != nil {
					return fmt.Errorf("error processing PDF %s: %w", path, err)
				}
				mu.Lock()
				allStats = append(allStats, localStats)
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		exitWithError(err)
	}

	if len(allStats) == 0 {
		fmt.Println("No results to display as no PDFs were processed.")
		os.Exit(0)
	}

	fmt.Println("Name,White Pixels,Non-WhitePixels,Percentage White Pixels")

	for _, stats := range allStats {
		fmt.Printf("%s,%d,%d,%f\n",
			stats.name, stats.whitePixels, stats.nonWhitePixels, stats.percentageWhitePixels)
	}
}

func exitWithError(err error) {
	log.Fatalf("%s", err)
}

func forEachPdf(source string, action func(string)) error {
	if strings.HasSuffix(source, ".pdf") {
		action(source)
		return nil
	}

	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pdf") {
			action(filepath.Join(source, entry.Name()))
		}
	}

	return nil
}

func processPdf(tempDir string, path string) (*pdfStats, error) {
	doc, err := fitz.New(path)
	if err != nil {
		return nil, err
	}

	defer doc.Close()

	opts := &jpeg.Options{Quality: jpeg.DefaultQuality}
	var whitePixels int64
	var nonWhitePixels int64
	name := filepath.Base(path)

	for n := 0; n < doc.NumPage(); n++ {
		img, err := doc.Image(n)
		if err != nil {
			return nil, err
		}

		f, err := os.Create(filepath.Join(tempDir, fmt.Sprintf("%s-%03d.jpeg", name, n)))
		if err != nil {
			return nil, err
		}

		defer f.Close()

		err = jpeg.Encode(f, img, opts)
		if err != nil {
			return nil, err
		}

		f.Seek(0, 0)
		cfg, _, err := image.DecodeConfig(f)
		if err != nil {
			return nil, err
		}

		width := cfg.Width
		height := cfg.Height

		f.Seek(0, 0)
		decoded, _, err := image.Decode(f)
		if err != nil {
			return nil, err
		}

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				r, g, b, a := decoded.At(x, y).RGBA()
				if isWhite(r, g, b, a) {
					whitePixels++
				} else {
					nonWhitePixels++
				}
			}
		}
	}

	return &pdfStats{
		name:                  name,
		whitePixels:           whitePixels,
		nonWhitePixels:        nonWhitePixels,
		percentageWhitePixels: 100.0 * float64(whitePixels) / float64(whitePixels+nonWhitePixels),
	}, nil
}

func isWhite(r uint32, g uint32, b uint32, _ uint32) bool {
	rc := int(r / 257)
	gc := int(r / 257)
	bc := int(r / 257)

	return rc == 255 && gc == 255 && bc == 255
}
