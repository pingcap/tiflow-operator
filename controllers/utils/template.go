package utils

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

// todo: Need to complete the code for the executor startup script
var executorStartScriptTpl = template.Must(template.New("tiflow-executor-start-script").Parse(""))

type ExecutorStartScriptModel struct {
	DataDir       string
	MasterAddress string
}

func RenderExecutorStartScript(model *ExecutorStartScriptModel) (string, error) {
	return renderTemplateFunc(executorStartScriptTpl, model)
}
