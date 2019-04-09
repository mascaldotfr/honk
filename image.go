//
// Copyright (c) 2019 Ted Unangst <tedu@tedunangst.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
)

func lineate(s uint8) float64 {
	x := float64(s)
	x /= 255.0
	if x < 0.04045 {
		x /= 12.92
	} else {
		x += 0.055
		x /= 1.055
		x = math.Pow(x, 2.4)
	}
	return x
}

func delineate(x float64) uint8 {
	if x > 0.0031308 {
		x = math.Pow(x, 1/2.4)
		x *= 1.055
		x -= 0.055
	} else {
		x *= 12.92
	}
	x *= 255.0
	return uint8(x)
}

func blend(d []byte, s1, s2, s3, s4 int) byte {
	l1 := lineate(d[s1])
	l2 := lineate(d[s2])
	l3 := lineate(d[s3])
	l4 := lineate(d[s4])
	return delineate((l1 + l2 + l3 + l4) / 4.0)
}

func squish(d []byte, s1, s2, s3, s4 int) byte {
	return uint8((uint32(s1) + uint32(s2)) / 2)
}

func vacuumwrap(img image.Image, format string) ([]byte, string, error) {
	maxdimension := 2048
	for img.Bounds().Max.X > maxdimension || img.Bounds().Max.Y > maxdimension {
		switch oldimg := img.(type) {
		case *image.NRGBA:
			w, h := oldimg.Rect.Max.X/2, oldimg.Rect.Max.Y/2
			newimg := image.NewNRGBA(image.Rectangle{Max: image.Point{X: w, Y: h}})
			for j := 0; j < h; j++ {
				for i := 0; i < w; i++ {
					p := newimg.Stride*j + i*4
					q1 := oldimg.Stride*(j*2+0) + i*4*2
					q2 := oldimg.Stride*(j*2+1) + i*4*2
					newimg.Pix[p+0] = blend(oldimg.Pix, q1+0, q1+4, q2+0, q2+4)
					newimg.Pix[p+1] = blend(oldimg.Pix, q1+1, q1+5, q2+1, q2+5)
					newimg.Pix[p+2] = blend(oldimg.Pix, q1+2, q1+6, q2+2, q2+6)
					newimg.Pix[p+3] = squish(oldimg.Pix, q1+3, q1+7, q2+3, q2+7)
				}
			}
			img = newimg
		case *image.YCbCr:
			w, h := oldimg.Rect.Max.X/2, oldimg.Rect.Max.Y/2
			newimg := image.NewYCbCr(image.Rectangle{Max: image.Point{X: w, Y: h}},
				oldimg.SubsampleRatio)
			for j := 0; j < h; j++ {
				for i := 0; i < w; i++ {
					p := newimg.YStride*j + i
					q1 := oldimg.YStride*(j*2+0) + i*2
					q2 := oldimg.YStride*(j*2+1) + i*2
					newimg.Y[p+0] = blend(oldimg.Y, q1+0, q1+1, q2+0, q2+1)
				}
			}
			switch newimg.SubsampleRatio {
			case image.YCbCrSubsampleRatio444:
				w, h = w, h
			case image.YCbCrSubsampleRatio422:
				w, h = w/2, h
			case image.YCbCrSubsampleRatio420:
				w, h = w/2, h/2
			case image.YCbCrSubsampleRatio440:
				w, h = w, h/2
			case image.YCbCrSubsampleRatio411:
				w, h = w/4, h
			case image.YCbCrSubsampleRatio410:
				w, h = w/4, h/2
			}
			for j := 0; j < h; j++ {
				for i := 0; i < w; i++ {
					p := newimg.CStride*j + i
					q1 := oldimg.CStride*(j*2+0) + i*2
					q2 := oldimg.CStride*(j*2+1) + i*2
					newimg.Cb[p+0] = blend(oldimg.Cb, q1+0, q1+1, q2+0, q2+1)
					newimg.Cr[p+0] = blend(oldimg.Cr, q1+0, q1+1, q2+0, q2+1)
				}
			}
			img = newimg
		default:
			return nil, "", fmt.Errorf("can't support image format")
		}
	}
	maxsize := 512 * 1024
	quality := 80
	var buf bytes.Buffer
	for {
		switch format {
		case "png":
			png.Encode(&buf, img)
		case "jpeg":
			jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		default:
			return nil, "", fmt.Errorf("can't encode format: %s", format)
		}
		if buf.Len() > maxsize && quality > 30 {
			switch format {
			case "png":
				format = "jpeg"
			case "jpeg":
				quality -= 10
			}
			buf.Reset()
			continue
		}
		break
	}
	return buf.Bytes(), format, nil
}
