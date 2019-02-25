package main

import (
	"crypto/rsa"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type tokens struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

type tokenGeneratorFunc func(claims map[string]interface{}) (*tokens, error)

// mustNewTokenGenerator creates a token generator configured with asymmetric crypto/RSA keys
func mustNewTokenGenerator(privKeyBytes []byte) tokenGeneratorFunc {

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKeyBytes) //ParseRSAPrivateKeyFromPEMWithPassword
	fatal(err)

	return func(claims map[string]interface{}) (*tokens, error) {
		return createTokens(signKey, claims)
	}
}

func createTokens(signKey *rsa.PrivateKey, claims map[string]interface{}) (*tokens, error) {
	//TODO: configure these
	const iss string = "https://auth.hacknights.club" //issuer - the authorization server that issued the token (this service)
	const accessExp time.Duration = time.Minute * 1
	const refreshExp time.Duration = time.Hour * 24 * 7
	const aud string = "https://api.hacknights.club" //audience - the relaying party(s) that can use the token (applications)

	access := func(now time.Time) *token {
		//Custom Claims
		mc := jwt.MapClaims{}
		for k, v := range claims {
			mc[k] = v
		}

		//Standard Claims
		mc["iss"] = iss
		mc["aud"] = aud
		mc["typ"] = "access"
		mc["sub"] = claims["sub"] //subject - end user identifier
		mc["iat"] = now.Unix()
		mc["exp"] = now.Add(accessExp).Unix()

		return newToken(mc)
	}

	refresh := func(now time.Time) *token {

		mc := jwt.MapClaims{
			"iss": iss,
			"aud": iss,
			"typ": "refresh",
			"rev": claims["rev"], //revision - the last significant change
			"sub": claims["sub"],
			"iat": now.Unix(),
			"exp": now.Add(refreshExp).Unix(),
		}

		return newToken(mc)
	}

	//TODO: Signing is slow, so trying to do it in parallel - but this ugly!

	now := time.Now()

	a := access(now)
	go a.sign(signKey)

	r := refresh(now)
	go r.sign(signKey)

	t := &tokens{}
	for t.Access == "" || t.Refresh == "" {
		select {
		case t.Access = <-a.signed:
		case err := <-a.err:
			if err != nil {
				return nil, err
			}
		case t.Refresh = <-r.signed:
		case err := <-r.err:
			if err != nil {
				return nil, err
			}
		}
	}
	return t, nil
}

type token struct {
	*jwt.Token
	signed chan string
	err    chan error
}

func newToken(claims jwt.MapClaims) *token {
	//TODO: Populate the Header with the KeyID
	/*
		func NewWithClaims(method SigningMethod, claims Claims) *Token {
			return &Token{
				Header: map[string]interface{}{
					"typ": "JWT",
					"alg": method.Alg(),
				},
				Claims: claims,
				Method: method,
			}
		}
	*/
	return &token{
		Token:  jwt.NewWithClaims(jwt.SigningMethodRS512, claims),
		signed: make(chan string),
		err:    make(chan error),
	}
}

func (t *token) sign(signKey *rsa.PrivateKey) {
	signed, err := t.SignedString(signKey)
	t.signed <- signed
	t.err <- err
}
