package main

import (
	"os"
	"testing"
)

func TestPercentageWhitespace(t *testing.T) {
	tempDir, err := os.MkdirTemp(os.TempDir(), "pdf-whitespace-analyzer-tests")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	defer func() {
		os.RemoveAll(tempDir)
	}()

	t.Run("single page, 100% white pixels", func(t *testing.T) {
		stats, err := processPdf(tempDir, "./test_candidates/100w_0nw_1p.pdf")
		if err != nil {
			t.Log(err)
			t.Fail()
		}

		if stats.percentageWhitePixels < 100 {
			t.Logf("Expected %f percentage white pixels to be 100%%", stats.percentageWhitePixels)
			t.Fail()
		}
	})

	t.Run("single page, 50% white pixels, 50% non-white pixels", func(t *testing.T) {
		stats, err := processPdf(tempDir, "./test_candidates/50w_50nw_1p.pdf")
		if err != nil {
			t.Log(err)
			t.Fail()
		}

		if stats.percentageWhitePixels < 49.5 || stats.percentageWhitePixels > 50.5 {
			t.Logf("expected %f percentage white pixels to be between 49.5 and 50.5", stats.percentageWhitePixels)
			t.Fail()
		}
	})

	t.Run("multiple pages, 75% white pixels, 25% non-white pixels", func(t *testing.T) {
		stats, err := processPdf(tempDir, "./test_candidates/75w_25nw_2p.pdf")
		if err != nil {
			t.Log(err)
			t.Fail()
		}

		if stats.percentageWhitePixels < 74.5 || stats.percentageWhitePixels > 75.5 {
			t.Logf("expected %f percentage white pixels to be between 74.5 and 75.5", stats.percentageWhitePixels)
			t.Fail()
		}
	})
}
