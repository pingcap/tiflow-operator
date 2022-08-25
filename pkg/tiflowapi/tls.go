package tiflowapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetTLSConfig returns *tls.Config for given TiDB cluster.
func GetTLSConfig(cli client.Client, namespace, secretName string) (*tls.Config, error) {
	secret := &corev1.Secret{}
	err := cli.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}, secret)
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", namespace, secretName, err)
	}

	return LoadTlsConfigFromSecret(secret)
}

func LoadTlsConfigFromSecret(secret *corev1.Secret) (*tls.Config, error) {
	rootCAs := x509.NewCertPool()
	var tlsCert tls.Certificate

	if !rootCAs.AppendCertsFromPEM(secret.Data[corev1.ServiceAccountRootCAKey]) {
		return nil, fmt.Errorf("failed to append ca certs")
	}

	clientCert, certExists := secret.Data[corev1.TLSCertKey]
	clientKey, keyExists := secret.Data[corev1.TLSPrivateKeyKey]
	if !certExists || !keyExists {
		return nil, fmt.Errorf("cert or key does not exist in secret %s/%s", secret.Namespace, secret.Name)
	}
	tlsCert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", secret.Namespace, secret.Name, err)
	}

	return &tls.Config{
		RootCAs:      rootCAs,
		ClientCAs:    rootCAs,
		Certificates: []tls.Certificate{tlsCert},
	}, nil
}
