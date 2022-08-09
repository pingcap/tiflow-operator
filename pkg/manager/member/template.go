package member

import (
	"bytes"
	"text/template"
)

func renderTemplateFunc(tpl *template.Template, model interface{}) (string, error) {
	buff := new(bytes.Buffer)
	err := tpl.Execute(buff, model)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}

// tiflowMasterStartScriptTpl is the tiflow-master start script
// Note: changing this will cause a rolling-update of tiflow-master cluster
var tiflowMasterStartScriptTpl = template.Must(template.New("tiflow-master-start-script").Parse(`#!/bin/sh

# This script is used to start tiflow-master containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"

if [[ ! -f "${ANNOTATIONS}" ]]
then
    echo "${ANNOTATIONS} does't exist, exiting."
    exit 1
fi
source ${ANNOTATIONS} 2>/dev/null

runmode=${runmode:-normal}
if [[ X${runmode} == Xdebug ]]
then
    echo "entering debug mode."
    tail -f /dev/null
fi

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}

ARGS="--addr=:10240 \
--advertise-addr=${POD_NAME}.${PEER_SERVICE_NAME}:10240 \
--config=/etc/tiflow-master/tiflow-master.toml \
"

echo "starting tiflow-master ..."
sleep $((RANDOM % 10))
echo "/tiflow master ${ARGS}"
exec /tiflow master ${ARGS}
`))

type TiflowMasterStartScriptModel struct {
	Scheme string
}

func RenderTiflowMasterStartScript(model *TiflowMasterStartScriptModel) (string, error) {
	return renderTemplateFunc(tiflowMasterStartScriptTpl, model)
}

// tiflowExecutorStartScriptTpl is the tiflow-executor start script
var tiflwExecutorStartScriptTpl = template.Must(template.New("tiflow-executor-start-script").Parse(`#!/bin/sh

# This script is used to start tiflow-executor containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"

if [[ ! -f "${ANNOTATIONS}" ]]
then
    echo "${ANNOTATIONS} does't exist, exiting."
    exit 1
fi
source ${ANNOTATIONS} 2>/dev/null

runmode=${runmode:-normal}
if [[ X${runmode} == Xdebug ]]
then
    echo "entering debug mode."
    tail -f /dev/null
fi

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}

ARGS="--join={{ .MasterAddress }} \
--addr=:10241 \
--advertise-addr=${POD_NAME}.${PEER_SERVICE_NAME}:10241 \
--config={{ .DataDir }}/tiflow-executor.toml \
"

echo "starting tiflow-executor ..."
sleep $((RANDOM % 10))
echo "/tiflow executor ${ARGS}"
exec /tiflow executor ${ARGS}
`))

type TiflowExecutorStartScriptModel struct {
	DataDir       string
	MasterAddress string
}

func RenderExecutorStartScript(model *TiflowExecutorStartScriptModel) (string, error) {
	return renderTemplateFunc(tiflwExecutorStartScriptTpl, model)
}
