package middleware

import (
	"bytes"
	"crypto/rand"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// CaptchaType defines the type of captcha characters
type CaptchaType int

const (
	TypeNumeric      CaptchaType = iota // Numbers only
	TypeAlphabetic                      // Letters only
	TypeAlphanumeric                    // Letters and numbers
)

// CaptchaConfig defines the configuration for captcha
type CaptchaConfig struct {
	Length        int         // Captcha text length
	Width         int         // Image width
	Height        int         // Image height
	Type          CaptchaType // Captcha type
	NoiseLevel    int         // Noise level (0–100)
	ExpireTime    time.Duration
	SessionKey    string // Key to store captcha in session
	CaseSensitive bool   // Whether it is case sensitive
}

// DefaultCaptchaConfig returns the default configuration
func DefaultCaptchaConfig() CaptchaConfig {
	return CaptchaConfig{
		Length:        6,
		Width:         200,
		Height:        80,
		Type:          TypeAlphanumeric,
		NoiseLevel:    50,
		ExpireTime:    5 * time.Minute,
		SessionKey:    "captcha",
		CaseSensitive: false,
	}
}

// CaptchaStore stores captcha data
type CaptchaStore struct {
	mu       sync.RWMutex
	captchas map[string]captchaData
}

type captchaData struct {
	value      string
	expireTime time.Time
}

var store = &CaptchaStore{
	captchas: make(map[string]captchaData),
}

// GenerateCaptcha is a middleware to generate captcha
func GenerateCaptcha(config ...CaptchaConfig) gin.HandlerFunc {
	cfg := DefaultCaptchaConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Cleanup expired captchas periodically
	go cleanupExpiredCaptchas()

	return func(c *gin.Context) {
		// Generate random text
		text := generateRandomText(cfg.Length, cfg.Type)

		// Generate captcha ID
		captchaID := generateID()

		// Store captcha
		store.mu.Lock()
		store.captchas[captchaID] = captchaData{
			value:      text,
			expireTime: time.Now().Add(cfg.ExpireTime),
		}
		store.mu.Unlock()

		// Generate image
		img := generateCaptchaImage(text, cfg)

		// Encode to PNG
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate captcha"})
			return
		}

		// Set captcha ID in cookie or response header
		c.Header("X-Captcha-ID", captchaID)
		c.SetCookie("captcha_id", captchaID, int(cfg.ExpireTime.Seconds()), "/", "", false, true)

		// Return image
		c.Data(200, "image/png", buf.Bytes())
	}
}

// VerifyCaptcha is a middleware to verify captcha
func VerifyCaptcha(caseSensitive ...bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		captchaID, err := c.Cookie("captcha_id")
		if err != nil {
			captchaID = c.GetHeader("X-Captcha-ID")
		}

		if captchaID == "" {
			c.JSON(400, gin.H{"error": "Captcha ID not found"})
			c.Abort()
			return
		}

		userInput := c.PostForm("captcha")
		if userInput == "" {
			userInput = c.Query("captcha")
		}

		if userInput == "" {
			c.JSON(400, gin.H{"error": "Captcha value required"})
			c.Abort()
			return
		}

		// Verify captcha
		store.mu.RLock()
		data, exists := store.captchas[captchaID]
		store.mu.RUnlock()

		if !exists {
			c.JSON(400, gin.H{"error": "Invalid or expired captcha"})
			c.Abort()
			return
		}

		if time.Now().After(data.expireTime) {
			store.mu.Lock()
			delete(store.captchas, captchaID)
			store.mu.Unlock()
			c.JSON(400, gin.H{"error": "Captcha expired"})
			c.Abort()
			return
		}

		// Compare values
		isCaseSensitive := false
		if len(caseSensitive) > 0 {
			isCaseSensitive = caseSensitive[0]
		}

		valid := false
		if isCaseSensitive {
			valid = userInput == data.value
		} else {
			valid = equalIgnoreCase(userInput, data.value)
		}

		// Delete captcha after verification (one-time use)
		store.mu.Lock()
		delete(store.captchas, captchaID)
		store.mu.Unlock()

		if !valid {
			c.JSON(400, gin.H{"error": "Invalid captcha"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// generateRandomText creates random text based on the type
func generateRandomText(length int, captchaType CaptchaType) string {
	var charset string

	switch captchaType {
	case TypeNumeric:
		charset = "0123456789"
	case TypeAlphabetic:
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	case TypeAlphanumeric:
		charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}

	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

// generateID creates a random ID
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return string(b)
}

// generateCaptchaImage creates a captcha image with noise
func generateCaptchaImage(text string, cfg CaptchaConfig) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))

	// Background
	bgColor := color.RGBA{255, 255, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Add noise lines
	addNoiseLines(img, cfg)

	// Add noise dots
	addNoiseDots(img, cfg)

	// Draw text
	drawText(img, text, cfg)

	return img
}

// addNoiseLines adds random noise lines
func addNoiseLines(img *image.RGBA, cfg CaptchaConfig) {
	numLines := cfg.NoiseLevel / 10
	for i := 0; i < numLines; i++ {
		x1, _ := rand.Int(rand.Reader, big.NewInt(int64(cfg.Width)))
		y1, _ := rand.Int(rand.Reader, big.NewInt(int64(cfg.Height)))
		x2, _ := rand.Int(rand.Reader, big.NewInt(int64(cfg.Width)))
		y2, _ := rand.Int(rand.Reader, big.NewInt(int64(cfg.Height)))

		r, _ := rand.Int(rand.Reader, big.NewInt(256))
		g, _ := rand.Int(rand.Reader, big.NewInt(256))
		b, _ := rand.Int(rand.Reader, big.NewInt(256))

		lineColor := color.RGBA{uint8(r.Int64()), uint8(g.Int64()), uint8(b.Int64()), 200}
		drawLine(img, int(x1.Int64()), int(y1.Int64()), int(x2.Int64()), int(y2.Int64()), lineColor)
	}
}

// addNoiseDots adds random noise dots
func addNoiseDots(img *image.RGBA, cfg CaptchaConfig) {
	numDots := cfg.NoiseLevel * 5
	for i := 0; i < numDots; i++ {
		x, _ := rand.Int(rand.Reader, big.NewInt(int64(cfg.Width)))
		y, _ := rand.Int(rand.Reader, big.NewInt(int64(cfg.Height)))

		r, _ := rand.Int(rand.Reader, big.NewInt(256))
		g, _ := rand.Int(rand.Reader, big.NewInt(256))
		b, _ := rand.Int(rand.Reader, big.NewInt(256))

		dotColor := color.RGBA{uint8(r.Int64()), uint8(g.Int64()), uint8(b.Int64()), 150}
		img.Set(int(x.Int64()), int(y.Int64()), dotColor)
	}
}

// drawLine draws a line on the image
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx := math.Abs(float64(x2 - x1))
	dy := math.Abs(float64(y2 - y1))
	sx, sy := 1, 1
	if x1 >= x2 {
		sx = -1
	}
	if y1 >= y2 {
		sy = -1
	}
	err := dx - dy

	for {
		img.Set(x1, y1, c)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// drawText draws text onto the image
func drawText(img *image.RGBA, text string, cfg CaptchaConfig) {
	textColor := color.RGBA{0, 0, 0, 255}
	point := fixed.Point26_6{
		X: fixed.Int26_6((cfg.Width / (cfg.Length + 1)) * 64),
		Y: fixed.Int26_6((cfg.Height / 2) * 64),
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: basicfont.Face7x13,
		Dot:  point,
	}

	spacing := cfg.Width / (cfg.Length + 1)

	for i, char := range text {
		// Random vertical offset for each character
		offset, _ := rand.Int(rand.Reader, big.NewInt(20))
		yOffset := int(offset.Int64()) - 10

		d.Dot.X = fixed.Int26_6((spacing * (i + 1)) * 64)
		d.Dot.Y = fixed.Int26_6((cfg.Height/2 + yOffset) * 64)

		d.DrawString(string(char))
	}
}

// cleanupExpiredCaptchas removes expired captchas periodically
func cleanupExpiredCaptchas() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		store.mu.Lock()
		now := time.Now()
		for id, data := range store.captchas {
			if now.After(data.expireTime) {
				delete(store.captchas, id)
			}
		}
		store.mu.Unlock()
	}
}

// equalIgnoreCase compares two strings ignoring case sensitivity
func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if toLower(a[i]) != toLower(b[i]) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
