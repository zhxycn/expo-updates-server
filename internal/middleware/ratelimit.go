package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"golang.org/x/time/rate"
)

func RateLimit(limiter *rate.Limiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !limiter.Allow() {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "Too many requests.",
				})
			}
			return next(c)
		}
	}
}
