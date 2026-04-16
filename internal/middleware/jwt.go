package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

type JWT struct {
	Secret []byte
}

func (j *JWT) Generate(userId string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userId,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(j.Secret)
}

func (j *JWT) Auth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			auth := c.Request().Header.Get("Authorization")

			if !strings.HasPrefix(auth, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Missing or invalid Authorization header",
				})
			}

			token, err := jwt.Parse(strings.TrimPrefix(auth, "Bearer "), func(token *jwt.Token) (interface{}, error) {
				return j.Secret, nil
			})
			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Token invalid or expired",
				})
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid token claims",
				})
			}

			userId, _ := claims.GetSubject()
			c.Set("UserID", userId)

			return next(c)
		}
	}
}
