package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

type CookieConfig struct {
	Secure   bool
	HttpOnly bool
	SameSite string
	MaxAge   int
}

func isSecure() bool {
	return os.Getenv("ENVIRONMENT") == "production" || os.Getenv("ENV") == "production"
}

func SetSecureCookie(c *gin.Context, name, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isSecure(),
		SameSite: http.SameSiteLaxMode,
	})
}

func SetSecureCookieWithDomain(c *gin.Context, name, value string, maxAge int, domain string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   domain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isSecure(),
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSecureCookie(c *gin.Context, name string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure(),
		SameSite: http.SameSiteLaxMode,
	})
}
