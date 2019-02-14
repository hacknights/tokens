package main

import (
	"crypto/rsa"
	"io/ioutil"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
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

func createTokens(u *identity, signKey *rsa.PrivateKey) (string, string, error) {

	access := func() (string, error) {
		//Custom Claims
		mc := jwt.MapClaims{}
		for k, v := range u.Claims {
			mc[k] = v
		}

		//Standard Claims
		now := time.Now()
		mc["iss"] = "https://auth.tokens.com" //issuer - the authorization server that issued the token
		mc["aud"] = "https://api.devable.com" //audience - the relaying party(s) that can use the token (application)
		mc["sub"] = u.ID                      //subject - end user identifier
		mc["iat"] = now.Unix()
		mc["exp"] = now.Add(time.Minute * 1).Unix() //TODO: configurable expiration

		// create a signer for rsa 256
		t := jwt.NewWithClaims(jwt.SigningMethodRS512, mc)
		return t.SignedString(signKey)
	}

	refresh := func() (string, error) {
		now := time.Now()
		t := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.StandardClaims{
			Id:        uuid.New().String(),
			Issuer:    "https://auth.tokens.com",
			Audience:  "https://api.devable.com",
			Subject:   u.ID,
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(time.Hour * 24 * 7).Unix(), //TODO: configurable expiration
		})
		return t.SignedString(signKey)
	}

	a, err := access()
	if err != nil {
		return "", "", err
	}

	r, err := refresh()
	if err != nil {
		return "", "", err
	}

	return a, r, nil
}
