package protocol

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

// TLSConfigFromPEM builds a tls.Config from PEM cert/key.
func TLSConfigFromPEM(certPEM, keyPEM []byte, isServer bool) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	if isServer {
		cfg.ClientAuth = tls.RequestClientCert
	}
	return cfg, nil
}

// ListenAndServeTLS starts a TLS server.
func ListenAndServeTLS(addr string, cfg *tls.Config, onConn func(*TransportConnection)) (*TransportServer, error) {
	ln, err := tls.Listen("tcp", addr, cfg)
	if err != nil {
		return nil, err
	}
	ts := &TransportServer{listener: ln}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			// wrap net.Conn as TLS conn already; TransportConnection uses conn
			onConn(NewTransportConnection(c))
		}
	}()
	return ts, nil
}

// DialTLS connects over TLS.
func DialTLS(addr string, cfg *tls.Config) (*TransportConnection, error) {
	c, err := tls.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, err
	}
	return NewTransportConnection(c), nil
}

// Fingerprint of a cert (SHA-256 of DER).
func CertFingerprint(cert *x509.Certificate) string {
	return fmt.Sprintf("%x", sha256.Sum256(cert.Raw))
}