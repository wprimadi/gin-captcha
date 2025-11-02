# Gin Captcha Middleware

![Gin Captcha Middleware](https://raw.githubusercontent.com/wprimadi/gin-captcha/refs/heads/main/banner.png)

A flexible and customizable CAPTCHA middleware for Gin framework with noise effects and multiple character types support.

## Features

- 🔢 **Multiple Character Types**: Numeric, Alphabetic, or Alphanumeric
- 🎨 **Customizable Noise Effects**: Add random lines and dots for enhanced security
- ⚙️ **Flexible Configuration**: Customize length, size, noise level, and expiration time
- 🔒 **Secure**: Cryptographically secure random generation
- ⏰ **Auto-Expiration**: Automatic cleanup of expired captchas
- 🔄 **One-Time Use**: Captchas are deleted after verification
- 🎯 **Case-Sensitive Options**: Support for both case-sensitive and case-insensitive verification

## Installation

```bash
go get github.com/gin-gonic/gin
go get golang.org/x/image/font
go get golang.org/x/image/math/fixed
go get github.com/wprimadi/gin-captcha
```

## Quick Start

```go
package main

import (
    "time"
    "github.com/gin-gonic/gin"
    "github.com/wprimadi/gin-captcha"
)

func main() {
    r := gin.Default()

    // Generate captcha endpoint
    r.GET("/captcha", middleware.GenerateCaptcha())

    // Protected route with captcha verification
    r.POST("/submit", middleware.VerifyCaptcha(), func(c *gin.Context) {
        c.JSON(200, gin.H{
            "message": "Captcha valid! Form submitted successfully",
        })
    })

    r.Run(":8080")
}
```

## Configuration Options

```go
type CaptchaConfig struct {
    Length        int           // Length of captcha text (default: 6)
    Width         int           // Image width in pixels (default: 200)
    Height        int           // Image height in pixels (default: 80)
    Type          CaptchaType   // Character type (default: TypeAlphanumeric)
    NoiseLevel    int           // Noise level 0-100 (default: 50)
    ExpireTime    time.Duration // Expiration time (default: 5 minutes)
    SessionKey    string        // Session key name (default: "captcha")
    CaseSensitive bool          // Case sensitive verification (default: false)
}
```

## Captcha Types

```go
middleware.TypeNumeric      // Numbers only: 0-9
middleware.TypeAlphabetic   // Letters only: A-Z, a-z
middleware.TypeAlphanumeric // Letters and numbers: 0-9, A-Z, a-z
```

## Usage Examples

### Basic Usage with Default Configuration

```go
r.GET("/captcha", middleware.GenerateCaptcha())
```

### Numeric Only Captcha (4 digits)

```go
r.GET("/captcha/numeric", middleware.GenerateCaptcha(middleware.CaptchaConfig{
    Length:     4,
    Width:      150,
    Height:     60,
    Type:       middleware.TypeNumeric,
    NoiseLevel: 30,
    ExpireTime: 3 * time.Minute,
}))
```

### Alphabetic Only with High Noise

```go
r.GET("/captcha/alpha", middleware.GenerateCaptcha(middleware.CaptchaConfig{
    Length:        6,
    Width:         200,
    Height:        80,
    Type:          middleware.TypeAlphabetic,
    NoiseLevel:    80,
    ExpireTime:    5 * time.Minute,
    CaseSensitive: true,
}))
```

### Custom Alphanumeric Captcha

```go
r.GET("/captcha/custom", middleware.GenerateCaptcha(middleware.CaptchaConfig{
    Length:     8,
    Width:      250,
    Height:     100,
    Type:       middleware.TypeAlphanumeric,
    NoiseLevel: 60,
    ExpireTime: 10 * time.Minute,
}))
```

## Verification

### Case-Insensitive Verification (Default)

```go
r.POST("/submit", middleware.VerifyCaptcha(), func(c *gin.Context) {
    // Your handler logic
    c.JSON(200, gin.H{"message": "Success"})
})
```

### Case-Sensitive Verification

```go
r.POST("/submit", middleware.VerifyCaptcha(true), func(c *gin.Context) {
    // Your handler logic
    c.JSON(200, gin.H{"message": "Success"})
})
```

## HTML Form Example

```html
<!DOCTYPE html>
<html>
<head>
    <title>Captcha Form</title>
</head>
<body>
    <form method="POST" action="/submit">
        <div>
            <img src="/captcha" id="captcha-img" alt="captcha">
            <button type="button" onclick="refreshCaptcha()">Refresh</button>
        </div>
        <div>
            <input type="text" name="captcha" placeholder="Enter captcha" required>
        </div>
        <button type="submit">Submit</button>
    </form>

    <script>
        function refreshCaptcha() {
            document.getElementById('captcha-img').src = '/captcha?t=' + new Date().getTime();
        }
    </script>
</body>
</html>
```

## API Usage

The middleware automatically handles captcha ID through cookies and headers. When a user requests a captcha:

1. The server generates a random text based on configuration
2. Creates an image with noise effects
3. Stores the captcha value with an expiration time
4. Returns the image and sets a cookie with the captcha ID

For verification:

1. User submits the form with the captcha value in the `captcha` field
2. Middleware retrieves the captcha ID from cookie or header
3. Compares user input with stored value
4. Deletes the captcha (one-time use)
5. Allows or denies the request based on verification result

## Error Responses

The middleware returns the following error responses:

- `400 Bad Request`: Captcha ID not found, captcha value required, invalid or expired captcha
- `500 Internal Server Error`: Failed to generate captcha image

## Security Features

- **Cryptographically Secure Random**: Uses `crypto/rand` for generating random text
- **One-Time Use**: Captchas are automatically deleted after verification
- **Auto-Expiration**: Expired captchas are cleaned up automatically
- **Noise Effects**: Multiple noise layers make OCR attacks more difficult
- **Random Character Positioning**: Each character has random vertical offset

## Performance Considerations

- Captcha images are generated on-the-fly
- In-memory storage with automatic cleanup
- Background goroutine for expired captcha cleanup runs every minute
- No external dependencies for storage

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License

## Author

Wahyu Primadi
saya@wahyuprimadi.com
https://wahyuprimadi.com

## Acknowledgments

- Built with [Gin Web Framework](https://github.com/gin-gonic/gin)
- Uses [Go's image package](https://pkg.go.dev/image) for image generation
- Font rendering with [golang.org/x/image/font](https://pkg.go.dev/golang.org/x/image/font)