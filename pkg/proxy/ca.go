package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

func GetCAPaths() (string, string) {
	usr, _ := user.Current()
	return filepath.Join(usr.HomeDir, "ca.crt"), filepath.Join(usr.HomeDir, "ca.key")
}

func LoadOrCreateCA() (*x509.Certificate, *rsa.PrivateKey, error) {
	crtPath, keyPath := GetCAPaths()

	if _, err := os.Stat(crtPath); err == nil {
		// Load existing
		crtBytes, _ := os.ReadFile(crtPath)
		keyBytes, _ := os.ReadFile(keyPath)

		block, _ := pem.Decode(crtBytes)
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, nil, err
		}

		block, _ = pem.Decode(keyBytes)
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, err
		}

		return cert, key, nil
	}

	// Create new
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Autoscout AI Fleet CA"},
			CommonName:   "Autoscout Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	// Save Cert
	f, _ := os.Create(crtPath)
	pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	f.Close()

	// Save Key
	f, _ = os.Create(keyPath)
	pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	f.Close()

	cert, _ := x509.ParseCertificate(derBytes)
	return cert, priv, nil
}
