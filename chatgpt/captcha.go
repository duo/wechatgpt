// Credits to https://github.com/rodjunger/chatgptauth
package chatgpt

import (
	"bytes"
	b64 "encoding/base64"
	"errors"
	"image"
	"image/png"
	"os"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

type Captcha string

// toPng returns the captcha converted to png in []byte format
func (c Captcha) ToPng() ([]byte, error) {
	// Thanks usual human
	// https://stackoverflow.com/questions/42993407/how-to-create-and-export-svg-to-png-jpeg
	if len(c) == 0 {
		return nil, errors.New("toPng: empty captcha")
	}
	decoded, err := b64.StdEncoding.DecodeString(string(c)[26:])

	if err != nil {
		return nil, err
	}

	icon, _ := oksvg.ReadIconStream(bytes.NewReader(decoded))

	w := int(icon.ViewBox.W) * 5 // Make it big
	h := int(icon.ViewBox.H) * 5

	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	icon.Draw(rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())), 1)

	out := new(bytes.Buffer)

	err = png.Encode(out, rgba)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// ToFile converts the captcha to png and writes in to disk
func (c Captcha) ToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	png, err := c.ToPng()

	if err != nil {
		return err
	}

	_, err = f.Write(png)

	if err != nil {
		return err
	}

	return nil
}

// Simple helper to check if there's a captcha
func (c Captcha) Available() bool {
	return c != ""
}
