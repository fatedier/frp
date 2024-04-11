package auth

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
)

type JWTAuthSetterVerifier struct {
	additionalAuthScopes []v1.AuthScope
	token                string
	secret               string
}

func NewJWTAuth(additionalAuthScopes []v1.AuthScope, token, secret string) *JWTAuthSetterVerifier {
	return &JWTAuthSetterVerifier{
		additionalAuthScopes: additionalAuthScopes,
		token:                token,
		secret:               secret,
	}
}

func (auth *JWTAuthSetterVerifier) SetLogin(loginMsg *msg.Login) error {
	loginMsg.PrivilegeKey = auth.token
	return nil
}

func (auth *JWTAuthSetterVerifier) SetPing(pingMsg *msg.Ping) error {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	pingMsg.Timestamp = time.Now().Unix()
	pingMsg.PrivilegeKey = auth.token
	return nil
}

func (auth *JWTAuthSetterVerifier) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) error {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	newWorkConnMsg.Timestamp = time.Now().Unix()
	newWorkConnMsg.PrivilegeKey = auth.token
	return nil
}

func (auth *JWTAuthSetterVerifier) VerifyLogin(m *msg.Login) error {
	if m.User == "" {
		return errors.New("user is empty")
	}
	token := m.PrivilegeKey
	return auth.VerifyToken(m.User, token)
}

func (auth *JWTAuthSetterVerifier) VerifyPing(m *msg.Ping) error {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	token := m.PrivilegeKey
	return auth.VerifyToken("", token)
}

func (auth *JWTAuthSetterVerifier) VerifyNewWorkConn(m *msg.NewWorkConn) error {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	token := m.PrivilegeKey
	return auth.VerifyToken("", token)
}

func (auth *JWTAuthSetterVerifier) VerifyToken(user, token string) error {
	methodKey := map[string]string{jwt.SigningMethodHS256.Alg(): auth.secret}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	parsedToken, err := parser.Parse(token, func(t *jwt.Token) (any, error) {
		key, ok := methodKey[t.Method.Alg()]
		if !ok {
			return nil, fmt.Errorf("method %s is not supported", t.Method)
		}
		return []byte(key), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return errors.New("token is expired")
		}
		return err
	}

	if !parsedToken.Valid {
		return fmt.Errorf("token %s is invalid", token)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("claims %v is invalid", parsedToken.Claims)
	}

	if len(user) > 0 {
		id, found := claims["email"]
		if !found {
			id, _ = claims["id"]
		}
		if id != user {
			return fmt.Errorf("token %s is not for user %s", token, user)
		}
	}

	return nil
}
