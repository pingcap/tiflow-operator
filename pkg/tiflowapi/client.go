package tiflowapi

import (
	"crypto/tls"
	"fmt"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/pkg/util"
)

// GetMasterClient provides a MasterClient of real tiflow-master cluster
// podName == "": get load balancer service of tiflow-master cluster
// podNAme != "": get exact pod's service of tiflow-master client
func GetMasterClient(cli client.Client, namespace, tcName, podName string, tlsEnabled bool, hetero Heterogeneous) MasterClient {
	var scheme = "http"

	var tlsConfig *tls.Config
	var err error
	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = GetTLSConfig(cli, namespace, util.ClusterClientTLSSecretName(tcName))
		if err != nil {
			klog.Errorf("Unable to get tls config for tiflow cluster %q, master client may not work: %v", tcName, err)
			return NewMasterClient(MasterClientURL(namespace, tcName, podName, scheme, hetero), DefaultTimeout, tlsConfig)
		}

		return NewMasterClient(MasterClientURL(namespace, tcName, podName, scheme, hetero), DefaultTimeout, tlsConfig)
	}

	return NewMasterClient(MasterClientURL(namespace, tcName, podName, scheme, hetero), DefaultTimeout, nil)
}

type Heterogeneous struct {
	Hetero    bool
	Namespace string
	Name      string
}

// MasterClientURL builds the url of master client
func MasterClientURL(namespace, clusterName, podName, scheme string, hetero Heterogeneous) string {
	peer := ""
	if len(podName) > 0 {
		podName += "."
		peer = "-peer"
	}

	if hetero.Hetero {
		return fmt.Sprintf("%s://%s-tiflow-master.%s.svc:10240", scheme, hetero.Name, hetero.Namespace)
	}

	return fmt.Sprintf("%s://%s%s-tiflow-master%s.%s.svc:10240", scheme, podName, clusterName, peer, namespace)
}
