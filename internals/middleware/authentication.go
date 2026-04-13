package middleware

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
)

func AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Authorization value from header
		tokenstring := r.Header.Get("Authorization")
		// Get secret key from .env file
		err := godotenv.Load("../../variables.env")
		if err != nil {
			http.Error(w, "Token could not be authorized!", http.StatusUnauthorized)
			return
		}
		env_variables, err := godotenv.Read("../../variables.env")
		if err != nil {
			http.Error(w, "Token could not be authorized!", http.StatusUnauthorized)
			return
		}
		secret_key := env_variables["jwt_key"]
		// Validate token
		token, err := jwt.Parse(tokenstring, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret_key), nil
		})
		if err != nil {
			http.Error(w, "Token could not be authorized!", http.StatusUnauthorized)
			return
		}
		type contextKeyType string
		const contextKey contextKeyType = "key"
		// Put token claims inside the
		ctx := context.WithValue(r.Context(), contextKey, token.Claims.(jwt.MapClaims))
		r = r.WithContext(ctx)
		// Call next handler
		next.ServeHTTP(w, r)
	})
}
