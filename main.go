package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"os"
	"path/filepath"
	"reflect"
	"sigs.k8s.io/yaml"
	"strings"
	"text/template"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "<nothing>"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func NewTemplate(baseName string, valueMap map[string]string) *template.Template {
	t := template.New(baseName)
	sprigFns := sprig.TxtFuncMap()

	var klFuncs template.FuncMap = map[string]any{}
	klFuncs["include"] = func(templateName string, templateData any) (string, error) {
		buf := bytes.NewBuffer(nil)
		if err := t.ExecuteTemplate(buf, templateName, templateData); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	klFuncs["toYAML"] = func(txt any) (string, error) {
		a, ok := sprigFns["toPrettyJson"].(func(any) string)
		if !ok {
			panic("could not convert sprig.TxtFuncMap[toPrettyJson] into func(any) string")
		}
		ys, err := yaml.JSONToYAML([]byte(a(txt)))
		if err != nil {
			return "", err
		}
		return string(ys), nil
	}

	klFuncs["ENDL"] = func() string {
		return "\n"
	}

	klFuncs["K8sAnnotation"] = func(cond any, key string, value any) string {
		if cond == reflect.Zero(reflect.TypeOf(cond)).Interface() {
			return ""
		}
		return fmt.Sprintf("%s: \"%v\"", key, value)
	}
	klFuncs["K8sLabel"] = klFuncs["K8sAnnotation"]

	klFuncs["val"] = func(key string, defaultVal ...any) any {
		if x, ok := valueMap[key]; ok {
			return x
		}
		return defaultVal[0]
	}

	return t.Funcs(sprigFns).Funcs(klFuncs)
}

func main() {
	var setArgs arrayFlags
	flag.Var(&setArgs, "set", "--set key1=value1")
	tFile := flag.String("template", "", "--template <path-to-file>")
	flag.Parse()

	if *tFile == "" {
		panic(fmt.Errorf("bad template file path"))
	}

	valueMap := map[string]string{}
	for _, s := range os.Environ() {
		split := strings.Split(s, "=")
		valueMap[split[0]] = split[1]
	}
	for _, item := range setArgs {
		split := strings.Split(item, "=")
		valueMap[split[0]] = split[1]
	}

	baseName := filepath.Base(*tFile)
	t := NewTemplate(baseName, valueMap)
	_, err := t.ParseFiles(*tFile)
	if err != nil {
		fmt.Println("ERROR occurred: ", err)
		os.Exit(1)
	}

	if err := t.Execute(os.Stdout, valueMap); err != nil {
		fmt.Println("ERROR occurred: ", err)
	}
}
