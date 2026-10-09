package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired     = errors.New("token已过期")
	ErrTokenInvalid     = errors.New("token无效")
	ErrTokenMalformed   = errors.New("token格式错误")
	ErrTokenNotValidYet = errors.New("token尚未生效")
)

// Claims JWT声明
type Claims struct {
	UserID   int64  `json:"user_id"`
	CorpID   int64  `json:"corp_id"`
	CorpName string `json:"corp_name"`
	Username string `json:"username"`
	RoleType string `json:"role_type"`
	jwt.RegisteredClaims
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey    string
	ExpiresHours int
	Issuer       string
}

// JWT JWT工具
type JWT struct {
	config JWTConfig
}

// NewJWT 创建JWT实例
func NewJWT(config JWTConfig) *JWT {
	return &JWT{config: config}
}

// GenerateToken 生成token
func (j *JWT) GenerateToken(userID, corpID int64, corpName, username, roleType string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(j.config.ExpiresHours) * time.Hour)

	claims := Claims{
		UserID:   userID,
		CorpID:   corpID,
		CorpName: corpName,
		Username: username,
		RoleType: roleType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    j.config.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.config.SecretKey))
}

// ParseToken 解析token
func (j *JWT) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.config.SecretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, ErrTokenMalformed
		}
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotValidYet
		}
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// RefreshToken 刷新token
func (j *JWT) RefreshToken(tokenString string) (string, error) {
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		return "", err
	}

	return j.GenerateToken(claims.UserID, claims.CorpID, claims.CorpName, claims.Username, claims.RoleType)
}
