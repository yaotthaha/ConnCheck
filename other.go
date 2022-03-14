package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptoRand "crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"github.com/wumansgy/goEncrypt"
	mathRand "math/rand"
	"os/exec"
	"strings"
	"time"
)

func GenRandomString(n uint64) string {
	mathRand.Seed(time.Now().UnixNano())
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mathRand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TimeNow() *time.Time {
	t := time.Now()
	return &t
}

func GenEccKey() ([]byte, []byte, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), cryptoRand.Reader)
	if err != nil {
		return nil, nil, err
	}
	x509PrivateKey, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	block := pem.Block{
		Type:  "Yaott ECC PRIVATE KEY",
		Bytes: x509PrivateKey,
	}
	privateKeyOutput := bytes.NewBuffer(nil)
	if err = pem.Encode(privateKeyOutput, &block); err != nil {
		return nil, nil, err
	}
	x509PublicKey, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	publicBlock := pem.Block{
		Type:  "Yaott ECC PUBLIC KEY",
		Bytes: x509PublicKey,
	}
	publicKeyOutput := bytes.NewBuffer(nil)
	if err = pem.Encode(publicKeyOutput, &publicBlock); err != nil {
		return nil, nil, err
	}
	return publicKeyOutput.Bytes(), privateKeyOutput.Bytes(), nil
}

func EccEncrypt(plainText, key []byte) ([]byte, error) {
	block, _ := pem.Decode(key)
	tempPublicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	publicKey1 := tempPublicKey.(*ecdsa.PublicKey)
	publicKey := goEncrypt.ImportECDSAPublic(publicKey1)
	cryptText, err := goEncrypt.Encrypt(cryptoRand.Reader, publicKey, plainText, nil, nil)
	if err != nil {
		return nil, err
	}
	return cryptText, err
}

func EccDecrypt(cryptText, key []byte) ([]byte, error) {
	block, _ := pem.Decode(key)
	tempPrivateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	privateKey := goEncrypt.ImportECDSA(tempPrivateKey)
	plainText, err := privateKey.Decrypt(cryptText, nil, nil)
	if err != nil {
		return nil, err
	}
	return plainText, nil
}

func Base64Encode(Data []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(Data))
}

func Base64Decode(DataRaw []byte) ([]byte, error) {
	sDec, err := base64.StdEncoding.DecodeString(string(DataRaw))
	if err != nil {
		return nil, err
	}
	return sDec, nil
}

func CommandRun(Command ...string) {
	Cmd := exec.Command(ServerTerminal, ServerTerminalArg, strings.Join(Command, " "))
	Cmd.Stdout = nil
	Cmd.Stderr = nil
	err := Cmd.Run()
	if err != nil {
		Logout(2, "Run Fail:", Command)
	}
}
