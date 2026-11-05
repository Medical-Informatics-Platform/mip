package spawnapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
)

// writeCert writes a self-signed tls.crt / tls.key for 127.0.0.1 into dir.
func writeCert(dir, cn string) *x509.Certificate {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	Expect(err).NotTo(HaveOccurred())
	keyDER, err := x509.MarshalECPrivateKey(key)
	Expect(err).NotTo(HaveOccurred())
	// Key first: the watcher reloads on the cert write and needs a matching key.
	Expect(os.WriteFile(filepath.Join(dir, "tls.key"), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(dir, "tls.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600)).To(Succeed())
	cert, err := x509.ParseCertificate(der)
	Expect(err).NotTo(HaveOccurred())
	return cert
}

var _ = Describe("Listener", func() {
	It("serves TLS 1.2+ and picks up a renewed certificate", func() {
		dir := GinkgoT().TempDir()
		first := writeCert(dir, "first")
		certs, err := certwatcher.New(filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key"))
		Expect(err).NotTo(HaveOccurred())
		certs = certs.WithWatchInterval(100 * time.Millisecond)

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		addr := ln.Addr().String()
		Expect(ln.Close()).To(Succeed())

		srv := NewServer(k8sClient, fakeAuth{}, 5, 20, GinkgoLogr)
		l := &Listener{Addr: addr, Handler: srv.Handler(), GetCertificate: certs.GetCertificate}
		runCtx, stop := context.WithCancel(ctx)
		defer stop()
		go func() { _ = certs.Start(runCtx) }()
		go func() { _ = l.Start(runCtx) }()
		Eventually(func() error { return l.ReadyCheck(nil) }).Should(Succeed())

		trusting := func(cert *x509.Certificate, version uint16) *tls.Config {
			pool := x509.NewCertPool()
			pool.AddCert(cert)
			return &tls.Config{RootCAs: pool, MinVersion: version, MaxVersion: version}
		}
		served := func(cfg *tls.Config) (string, error) {
			conn, err := tls.Dial("tcp", addr, cfg)
			if err != nil {
				return "", err
			}
			defer conn.Close()
			return conn.ConnectionState().PeerCertificates[0].Subject.CommonName, nil
		}
		Expect(served(trusting(first, tls.VersionTLS12))).To(Equal("first"))
		_, err = served(trusting(first, tls.VersionTLS11))
		Expect(err).To(HaveOccurred())

		second := writeCert(dir, "second")
		Eventually(func() (string, error) { return served(trusting(second, tls.VersionTLS13)) }).
			WithTimeout(5 * time.Second).Should(Equal("second"))

		hc := &http.Client{Transport: &http.Transport{TLSClientConfig: trusting(second, tls.VersionTLS13)}}
		resp, err := hc.Get("https://" + addr + "/v1/namespaces/federation-x/servers/jupyter-alice")
		Expect(err).NotTo(HaveOccurred())
		resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

		stop()
		Eventually(func() error { return l.ReadyCheck(nil) }).ShouldNot(Succeed())
	})
})
