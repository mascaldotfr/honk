package main

import (
	"bytes"
	"crypto/sha512"
	"image"
	"image/png"
)

func avatar(name string) []byte {
	h := sha512.New()
	h.Write([]byte(name))
	s := h.Sum(nil)
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for i := 0; i < 64; i++ {
		for j := 0; j < 64; j++ {
			p := i*img.Stride + j*4
			xx := i/16*16 + j/16
			x := s[xx]
			if x < 64 {
				img.Pix[p+0] = 32
				img.Pix[p+1] = 0
				img.Pix[p+2] = 64
				img.Pix[p+3] = 255
			} else if x < 128 {
				img.Pix[p+0] = 32
				img.Pix[p+1] = 0
				img.Pix[p+2] = 92
				img.Pix[p+3] = 255
			} else if x < 192 {
				img.Pix[p+0] = 64
				img.Pix[p+1] = 0
				img.Pix[p+2] = 128
				img.Pix[p+3] = 255
			} else {
				img.Pix[p+0] = 96
				img.Pix[p+1] = 0
				img.Pix[p+2] = 160
				img.Pix[p+3] = 255
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
