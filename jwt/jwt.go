// Tideland Go REST Server Library - JSON Web Token
//
// Copyright (C) 2016 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package jwt

//--------------------
// IMPORTS
//--------------------

import (
	"crypto"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"strings"
)

//--------------------
// CONST
//--------------------

//--------------------
// JSON Web Token
//--------------------

type JWT interface {
	fmt.Stringer
}

type jwtHeader struct {
	Type      string `json:"typ"`
	Algorithm string `json:"alg"`  
}

type jwt struct {
	payload   interface{}
	key		  Key
	algorithm Algorithm
	token	  string
}

// Encodes creates a JSON Web Token for the given payload
// based on key and algorithm.
func Encode(payload interface{}, key Key, algorithm Algorithm) (JWT, error) {
	jwt := &jwt{
		payload:   payload,
		key:	   key,
		algorithm: algorithm,
	}
	headerPart, err := marshallAndEncode(jwtHeader{"JWT", algorithm})
	if err != nil {
		return nil, errors.Annotate(err, ErrCannotEncode, errorMessages, "header")
	}
	payloadPart, err := marshallAndEncode(payload)
	if err != nil {
		return nil, errors.Annotate(err, ErrCannotEncode, errorMessages, "payload")
	}
	dataParts := headerPart + "." + payloadPart
	signaturePart, err := signAndEncode(dataParts, key, algorithm)
	if err != nil {
		return nil, errors.Annotate(err, ErrCannotEncode, errorMessages, "signature")
	}
	jwt.token = dataParts + "." + signaturePart
	return jwt, nil	
}

// Decode creates a token out of a string without verification. The passed
// payload is used for the unmarshalling of the payload part.
func Decode(token string, payload interface{}) (JWT, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New(ErrCannotDecode, errorMessages, "parts")
	}
	var header jwtHeader
	err := decodeAndUnmarshall(parts[0], &header)
	if err != nil {
		return nil, errors.Annotate(err, ErrCannotDecode, errorMessages, "header")
	}
	err = decodeAndUnmarshall(parts[1], payload)
	if err != nil {
		return nil, errors.Annotate(err, ErrCannotDecode, errorMessages, "payload")
	}
	return &jwt{
		payload:   payload,
		algorithm: header.Algorithm,
	}, nil
}

// Verify creates a token out of a string and varifies it against
// the passed key. Like in Decode() the payload is used for unmarshalling.
func Verify(token string, payload interface{}, key Key) (JWT, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New(ErrCannotVerify, errorMessages, "parts")
	}
	var header jwtHeader
	err := decodeAndUnmarshall(parts[0], &header)
	if err != nil {
		return nil, errors.Annotate(err, ErrCannotVerify, errorMessages, "header")
	}
	err := decodeAndVerify(parts, key, header.Algorithm)
	if err != nil {
		return nil, errors.Annotate(err, ErrCannotVerify, errorMessages, "signature")
	}
	err = decodeAndUnmarshall(parts[1], payload)
	if err != nil {
		return nil, errors.Annotate(err, ErrCannotVerify, errorMessages, "payload")
	}
	return &jwt{
		payload:   payload,
		key:       key,
		algorithm: header.Algorithm,
	}, nil
}

// Payload returns the payload of the token.
func (jwt *jwt) Payload() interface{} {
	return jwt.payload
}

// Key return the key of the token only when it is a result of encoding
// or verification.
func (jwt *jwt) Key() (Key, error) {
	if jwt.key == nil {
		return nil, errors.New(ErrNoKey, errorMessages)
	}
	return jwt.key
}

// Algorithm returns the algorithm of the token after encoding,
// decoding, or verification.
func (jwt *jwt) Algorithm() Algorithm {
	return jwt.algorithm
}

// String implements the stringer interface.
func (jwt *jwt) String() string {
	return jwt.token	
}

//--------------------
// PRIVATE HELPERS
//--------------------

// marshallAndEncode marshals the passed value to JSON and
// creates a BASE64 string out of it.
func marshallAndEncode(value interface{}) (string, error) {
	jsonValue, err := json.Marshall(value)
	if err != nil {
		return nil, errors.Annotate(ErrJSONMarshalling, errorMessages)
	}
	encoded := base64.RawURLEncoding.EncodeToString(jsonValue)
	return encoded, nil
}

// decodeAndUnmarshall decodes a BASE64 encoded JSON string and
// unmarshals it into the passed value.
func decodeAndUnmarshall(part string, value interface{}) error {
	decoded, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return errors.Annotate(err, ErrInvalidTokenPart, errorMessages)
	}
	err = json.Unmarshall(decoded, value)
	if err != nil {
		return errors.Annotate(err, ErrJSONUnmarshalling, errorMessages)
	}
	return nil
}

// signAndEncode creates the signature for the data part (header and
// payload) of the token using the passed key and algorithm. The result
// is then encoded to BASE64.
func signAndEncode(data []byte, key Key, algorithm Algorithm) (string, error) {
	sig := algorithm.Sign(data, key)
	encoded := base64.RawURLEncoding.EncodeToString(sig)
	return encoded, nil
}

// decodeAndVerify decodes a BASE64 encoded signature and verifies
// the correct signing of the data part (header and payload) using the
// passed key and algorithm.
func decodeAndVerify(parts []string, key Key], algorithm Algorithm) error {
	data := []byte(parts[0] + "." + parts[1])
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return errors.Annotate(err, ErrInvalidTokenPart, errorMessages)
	}
	return algorithm.Verify(data, sig, key)
}

// EOF