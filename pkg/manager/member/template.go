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

// tiflowMasterStartScriptTpl is the dm-master start script
// Note: changing this will cause a rolling-update of dm-master cluster
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

ARGS="--data-dir={{ .DataDir }} \
--name=${POD_NAME} \
--peer-urls={{ .Scheme }}://0.0.0.0:8291 \
--advertise-peer-urls={{ .Scheme }}://${POD_NAME}.${HEADLESS_SERVICE_NAME}:8291 \
--master-addr=:10240 \
--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}:10240 \
--config=/etc/tiflow-master/tiflow-master.toml \
"

if [[ -f {{ .DataDir }}/join ]]
then
# The content of the join file is:
#   demo-dm-master-0=http://demo-dm-master-0.demo-dm-master-peer.demo.svc:8291,demo-dm-master-1=http://demo-dm-master-1.demo-dm-master-peer.demo.svc:8291
# The --join args must be:
#   --join=http://demo-dm-master-0.demo-dm-master-peer.demo.svc:8261,http://demo-dm-master-1.demo-dm-master-peer.demo.svc:8261
join=` + "`" + `cat {{ .DataDir }}/join | sed -e 's/8291/10240/g' | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ","` + "`" + `
join=${join%,}
ARGS="${ARGS} --join=${join}"
elif [[ ! -d {{ .DataDir }}/member/wal ]]
then
result=""
POD_NUM=${POD_NAME##*-}
if [[ POD_NUM -eq "0" ]]; then
result="--initial-cluster=${POD_NAME}={{ .Scheme }}://${POD_NAME}.${HEADLESS_SERVICE_NAME}:8291"
else
POD_NUM=$(expr $POD_NUM - 1)
POD_PREFIX=${POD_NAME%-*}
result=$(seq ${POD_NUM} | xargs -I_ echo -n ",{{ .Scheme }}://${POD_PREFIX}-_.${HEADLESS_SERVICE_NAME}:10240")
result="--join={{ .Scheme }}://${POD_PREFIX}-0.${HEADLESS_SERVICE_NAME}:10240${result}"
fi
ARGS="${ARGS} ${result}"
fi

echo "starting tiflow-master ..."
sleep $((RANDOM % 10))
echo "/tiflow master ${ARGS}"
exec /tiflow master ${ARGS}
`))

type TiflowMasterStartScriptModel struct {
	Scheme  string
	DataDir string
}

func RenderTiflowMasterStartScript(model *TiflowMasterStartScriptModel) (string, error) {
	return renderTemplateFunc(tiflowMasterStartScriptTpl, model)
}
