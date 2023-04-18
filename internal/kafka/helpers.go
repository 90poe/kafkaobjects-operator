package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"regexp"
	"strings"
)

const (
	// PEMCertificateBlock has to match PEM certificate
	PEMCertificateBlock = `-----BEGIN CERTIFICATE-----(.*)-----END CERTIFICATE-----`
	// PEMPrivKeyBlock has to match PEM Key
	PEMPrivKeyBlock = `-----BEGIN PRIVATE KEY-----(.*)-----END PRIVATE KEY-----`
	// ReplacePattern to be replacing in Regexps, as strings.Replace is faster than regexp
	ReplacePattern = `(.*)`
)

// parseCerts would parse cert in strings
func parseCerts(caStr, privStr, certStr string) (*tls.Config, error) {
	caCert, err := normalizePEMblock(caStr, PEMCertificateBlock, ReplacePattern)
	if err != nil {
		return nil, fmt.Errorf("error parsing ca certs: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCert))

	pemCert, err := normalizePEMblock(certStr, PEMCertificateBlock, ReplacePattern)
	if err != nil {
		return nil, fmt.Errorf("error parsing pem certs: %w", err)
	}
	pemKey, err := normalizePEMblock(privStr, PEMPrivKeyBlock, ReplacePattern)
	if err != nil {
		return nil, fmt.Errorf("error parsing pem keys: %w", err)
	}
	tlsCert, err := tls.X509KeyPair([]byte(pemCert), []byte(pemKey))
	if err != nil {
		return nil, fmt.Errorf("could not load X509 key pair: %w", err)
	}

	tls := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{tlsCert},
		RootCAs:      caCertPool,
	}
	return tls, nil
}

func normalizePEMblock(pem, regStr, replacePattern string) (string, error) {
	reg := regexp.MustCompile(regStr)
	if !reg.MatchString(pem) {
		return pem, nil
	}
	subs := reg.FindStringSubmatch(pem)
	if len(subs) != 2 {
		return "", fmt.Errorf("not recognized PEM block")
	}
	pemNorm := strings.ReplaceAll(subs[1], " ", "\\n")
	pemNorm = strings.ReplaceAll(regStr, replacePattern, pemNorm)
	pemNorm = strings.ReplaceAll(pemNorm, "\\n", "\n")
	return pemNorm, nil
}
