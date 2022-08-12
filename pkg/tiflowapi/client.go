package tiflowapi

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetMasterClient provides a MasterClient of real tiflow-master cluster
// podName == "": get load balancer service of tiflow-master cluster
// podNAme != "": get exact pod's service of tiflow-master client
func GetMasterClient(cli client.Client, namespace, tcName, podName string, tlsEnabled bool) MasterClient {
	var scheme = "http"

	// TODO: support tls later
	//var tlsConfig *tls.Config
	//var err error
	//if tlsEnabled {
	//	scheme = "https"
	//	tlsConfig, err = GetTLSConfig(cli, namespace, util.TiflowClientTLSSecretName(tcName))
	//	if err != nil {
	//		klog.Errorf("Unable to get tls config for tiflow cluster %q, master client may not work: %v", tcName, err)
	//		return NewMasterClient(MasterClientURL(namespace, tcName, scheme), DefaultTimeout, tlsConfig, true)
	//	}
	//
	//	return NewMasterClient(MasterClientURL(namespace, tcName, scheme), DefaultTimeout, tlsConfig, true)
	//}

	return NewMasterClient(MasterClientURL(namespace, tcName, podName, scheme), DefaultTimeout, nil, true)
}

// MasterClientURL builds the url of master client
func MasterClientURL(namespace, clusterName, podName, scheme string) string {
	peer := ""
	if len(podName) > 0 {
		podName += "."
		peer = "-peer"
	}
	return fmt.Sprintf("%s://%s%s-tiflow-master%s.%s.svc:10240", scheme, podName, clusterName, peer, namespace)
}
