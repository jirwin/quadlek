package comics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/fogleman/gg"
)

const lineSpacing = 1.5

var imageCache = sync.Map{}

type Bubble struct {
	PosX   float64 `json:"x"`
	PosY   float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type Template struct {
	Name     string    `json:"name"`
	Width    float64   `json:"width"`
	Height   float64   `json:"height"`
	Bubbles  []*Bubble `json:"bubbles"`
	ImageURL string    `json:"image_url"`
	fontPath string    `json:"-"`
}

func (t *Template) String() string {
	out, _ := json.Marshal(t)
	return string(out)
}

func (t *Template) getBaseImg() (image.Image, error) {
	var imageBytes []byte

	cacheItem, ok := imageCache.Load(t.ImageURL)
	if ok {
		imageBytes, ok = cacheItem.([]byte)
		if !ok {
			return nil, fmt.Errorf("error: invalid data in image cache")
		}
	} else {
		resp, err := http.Get(t.ImageURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		imageBytes, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		imageCache.Store(t.ImageURL, imageBytes)
	}
	tempFile, err := ioutil.TempFile("/tmp", "comic-base-img")
	if err != nil {
		return nil, err
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, bytes.NewReader(imageBytes))
	if err != nil {
		return nil, err
	}

	return gg.LoadImage(tempFile.Name())
}

func (b *Bubble) setFontSize(dc *gg.Context, text, fontPath string) {
	fontSize := float64(1)
	renderedHeight := float64(0)

Outer:
	for renderedHeight < b.Height-(dc.FontHeight()) {
		dc.LoadFontFace(fontPath, fontSize)
		wrappedText := dc.WordWrap(text, b.Width)
		for _, t := range wrappedText {
			w, _ := dc.MeasureString(t)
			if w > b.Width {
				break Outer
			}
		}
		renderedHeight = dc.FontHeight() * lineSpacing * float64(len(wrappedText))
		fontSize++
	}

	dc.LoadFontFace(fontPath, fontSize-1)
}

func (t *Template) Render(text []string) ([]byte, error) {
	if len(text) < len(t.Bubbles) {
		return nil, fmt.Errorf("error: not enough text for the template")
	}

	baseImg, err := t.getBaseImg()
	if err != nil {
		return nil, err
	}

	dc := gg.NewContextForImage(baseImg)
	dc.SetRGB(0, 0, 0)

	for i, bubble := range t.Bubbles {
		bubble.setFontSize(dc, text[i], t.fontPath)
		dc.DrawStringWrapped(text[i], bubble.PosX, bubble.PosY, 0, 0, bubble.Width, lineSpacing, gg.AlignLeft)
	}

	buf := bytes.NewBuffer(nil)
	err = dc.EncodePNG(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func NewTemplate(templateUrl string, fontPath string) (*Template, error) {
	resp, err := http.Get(templateUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	templateBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	t := &Template{}
	err = json.Unmarshal(templateBytes, t)
	if err != nil {
		return nil, err
	}

	t.fontPath = fontPath

	return t, nil
}
