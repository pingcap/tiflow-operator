package create

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pingcap/tiflow-operator/api/config"
	"github.com/pingcap/tiflow-operator/e2e"
	"github.com/pingcap/tiflow-operator/pkg/testutil"
	testenv "github.com/pingcap/tiflow-operator/pkg/testutil/env"
	"github.com/pingcap/tiflow-operator/pkg/testutil/mysql"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	tiflowVersion = "latest"
	validImage    = "chunzhuli/dataflow"
)

// TestCreateTiflowCluster tests the creation of insecure cluster, and it should be successful.
func TestCreateTiflowCluster(t *testing.T) {
	// Test Creating an insecure cluster
	// No actions on the cluster just create it and
	// tear it down.

	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	e := testenv.CreateActiveEnvForTest()
	env := e.Start()
	defer e.Stop()

	sb := testenv.NewSandbox(t, env)
	sb.StartManager(t)

	configMap := &config.GenericConfig{}
	configBytes := []byte(fmt.Sprintf(`[framework-meta]
        schema = "example_framework"
        endpoints = ["mysql.%[1]s.svc:3306"]
        user = "root"
        password = "123456"
      [business-meta]
        schema = "example_business"
        endpoints = ["mysql.%[1]s.svc:3306"]
        user = "root"
        password = "123456"`, sb.Namespace))
	require.NoError(t, configMap.UnmarshalTOML(configBytes))
	builder := testutil.NewBuilder("basic").WithMasterReplica(3).WithExecutorReplica(3).
		WithMasterImage(validImage).WithExecutorImage(validImage).
		WithVersion(tiflowVersion).WithMasterConfig(configMap)

	steps := testutil.Steps{
		{
			Name: "creates meta mysql for tiflow cluster",
			Test: func(t *testing.T) {
				mysqlResources := mysql.CreateMySQLNodeCR()
				for _, resource := range mysqlResources {
					if pv, ok := resource.(*corev1.PersistentVolume); ok {
						getPV := pv.DeepCopy()
						err := sb.Get(getPV)
						if err != nil {
							if strings.Contains(err.Error(), "not found") {
								err = nil
							}
							require.NoError(t, err)
						} else {
							require.NoError(t, sb.Delete(getPV))
							require.Eventually(t, func() bool {
								err = sb.Get(getPV)
								return err != nil && strings.Contains(err.Error(), "not found")
							}, 5*time.Second, time.Second)
						}
					}
					require.NoError(t, sb.Create(resource))
				}
				testutil.RequireDeploymentToBeReadyEventuallyTimeout(t, sb, "mysql", e2e.CreateClusterTimeout)

				t.Log("Done with mysql cluster")
			},
		},
		{
			Name: "creates 3-node basic cluster",
			Test: func(t *testing.T) {
				createTime := metav1.NewTime(time.Unix(time.Now().Unix(), 0))
				require.NoError(t, sb.Create(builder.Cr()))
				testutil.RequireClusterToBeReadyEventuallyTimeout(t, sb, builder, createTime, e2e.CreateClusterTimeout)

				t.Log("Done with basic cluster")
			},
		},
	}
	steps.Run(t)
}
