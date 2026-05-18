package auth

import(
	"fmt"
	"time"
	"errors"
	"strings"
	"net/http"
	"crypto/rand"
	"encoding/hex"
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
)

func HashPassword(password string) (string, error){
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil{
		return "", err
	}

	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error){
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil{
		return false, err
	}

	return match, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error){
	claims := jwt.RegisteredClaims{
		Issuer: "chirpy-access",
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject: userID.String(),
	}

	byteSecret := []byte(tokenSecret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(byteSecret)

	if err != nil {
		fmt.Printf("Error signing token: %v\n", err)
		return "", err
	}

	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error){
	byteSecret := []byte(tokenSecret)
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return byteSecret, nil
	})

	if err != nil {
		fmt.Printf("Error procesing token: %v\n", err)
		return uuid.Nil, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)

	if !ok {
		return uuid.Nil, errors.New("unknown claims type, cannot proceed")
	}

	subject := claims.Subject
	userID, err := uuid.Parse(subject)

	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error){
	rep := strings.NewReplacer("Bearer ", "")
	token := headers.Get("Authorization")

	if token == ""{
		return "", errors.New("Authorization header not found")
	}

	token = rep.Replace(token)

	return token, nil
}

func MakeRefreshToken() string{
	key := make([]byte, 32)
	rand.Read(key)
	encodedStr := hex.EncodeToString(key)
	return encodedStr
}

func GetAPIKey(headers http.Header) (string, error){
	rep := strings.NewReplacer("ApiKey ", "")
	token := headers.Get("Authorization")

	if token == ""{
		return "", errors.New("Authorization header not found")
	}

	token = rep.Replace(token)

	return token, nil
}