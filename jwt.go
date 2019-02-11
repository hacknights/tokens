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

// User is a minimal interface for JWT
type User interface {
	ID() string
	Claims() map[string]interface{}
}

func createTokens(u User, signKey *rsa.PrivateKey) (string, string, error) {

	access := func() (string, error) {
		// create a signer for rsa 256
		t := jwt.New(jwt.GetSigningMethod("RS256"))

		//Custom Claims
		t.Claims = u.Claims()
		//Standard Claims
		now := time.Now()
		t.Claims["iss"] = "https://auth.authidate.com" //issuer - the authorization server that issued the token
		t.Claims["aud"] = "https://api.authidate.com"  //audience - the relaying party(s) that can use the token (application)
		t.Claims["sub"] = u.ID()                       //subject - end user identifier
		t.Claims["iat"] = now.Unix()
		t.Claims["exp"] = now.Add(time.Minute * 1).Unix() //TODO: configurable expiration
		return t.SignedString(signKey)
	}

	refresh := func() (string, error) {
		// create a signer for rsa 256
		t := jwt.New(jwt.GetSigningMethod("RS256"))
		//Standard Claims
		now := time.Now()
		t.Claims["jti"] = "uuid"
		t.Claims["iss"] = "https://auth.authidate.com" //issuer - the authorization server that issued the token
		t.Claims["sub"] = u.ID()                       //subject - end user identifier
		t.Claims["iat"] = now.Unix()
		t.Claims["exp"] = now.Add(time.Hour * 24 * 7).Unix() //TODO: configurable expiration
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
