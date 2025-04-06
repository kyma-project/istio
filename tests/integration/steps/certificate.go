package steps

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/testcontext"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/big"
	"time"
)

const (
	keySize         = 2048
	commonName      = "api-gateway-tests.example.com"
	certificateType = "CERTIFICATE"
	privateKeyType  = "PRIVATE KEY"
	tlsKeyName      = "tls.key"
	tlsCrtName      = "tls.crt"
)

func CreateDummySecretWithCert(ctx context.Context, name string, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	prvPEMBytes, crtPEMBytes, err := createDummyKeyAndCert()
	if err != nil {
		return ctx, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			tlsKeyName: prvPEMBytes,
			tlsCrtName: crtPEMBytes,
		},
		Type: corev1.SecretTypeTLS,
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), secret)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, secret)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func createDummyKeyAndCert() ([]byte, []byte, error) {
	prvKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, nil, err
	}
	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment

	notBefore := time.Now().AddDate(-1, 0, 0)
	notAfter := time.Now().AddDate(1, 0, 0)

	serialNumber := big.NewInt(1)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	crtDERBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &prvKey.PublicKey, prvKey)
	if err != nil {
		return nil, nil, err
	}

	crtPEMBytes := pem.EncodeToMemory(&pem.Block{Type: certificateType, Bytes: crtDERBytes})
	if crtPEMBytes == nil {
		return nil, nil, errors.New("failed to encode certificate")
	}

	prvDERBytes, err := x509.MarshalPKCS8PrivateKey(prvKey)
	if err != nil {
		return nil, nil, err
	}

	prvPEMBytes := pem.EncodeToMemory(&pem.Block{Type: privateKeyType, Bytes: prvDERBytes})
	if prvPEMBytes == nil {
		return nil, nil, errors.New("failed to encode private key")
	}

	return prvPEMBytes, crtPEMBytes, nil
}
