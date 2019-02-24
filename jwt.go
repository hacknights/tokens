package main

import (
	"crypto/rsa"
	"io/ioutil"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// location of the files used for signing and verification
const (
	privKeyPath = "keys/app.rsa"     // openssl genrsa -out app.rsa keysize
	pubKeyPath  = "keys/app.rsa.pub" // openssl rsa -in app.rsa -pubout > app.rsa.pub
)

// using asymmetric crypto/RSA keys
var (
	verifyKey *rsa.PublicKey
	signKey   *rsa.PrivateKey //TODO: Consider not holding this in memory
)

func init() {
	signBytes, err := ioutil.ReadFile(privKeyPath)
	fatal(err)

	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	fatal(err)

	verifyBytes, err := ioutil.ReadFile(pubKeyPath)
	fatal(err)

	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	fatal(err)
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type tokens struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

func createTokens(claims map[string]interface{}, signKey *rsa.PrivateKey) (*tokens, error) {
	//TODO: configure these
	const iss string = "https://auth.hacknights.club" //issuer - the authorization server that issued the token (this service)
	const aud string = "https://api.hacknights.club"  //audience - the relaying party(s) that can use the token (applications)
	const accessExp time.Duration = time.Minute * 1
	const refreshExp time.Duration = time.Hour * 24 * 7

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
			"aud": aud,
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
