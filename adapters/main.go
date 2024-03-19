package main

import (
	"flag"
	"os"
	"text/template"
)

func main() {
	tmpl := Template{}
	LoadDataFromFlag(&tmpl)
	WriteTemplate(tmpl)
}

func LoadDataFromFlag(tmpl *Template) {
	var pkg string
	var topic string
	var recv string
	var send string

	flag.StringVar(&pkg, "pkg", "main", "Package")
	flag.StringVar(&topic, "topic", "Topic", "Subject")
	flag.StringVar(&recv, "recv", "SubMsg", "RecvMessage Name")
	flag.StringVar(&send, "send", "PubMsg", "SendMessage Name")
	flag.Parse()

	tmpl.Package = pkg
	tmpl.Subject = topic
	tmpl.RecvMessage = recv
	tmpl.SendMessage = send
}

func WriteTemplate(tmpl Template) {
	t1 := template.Must(template.New("template").Parse(MuxTmpl))
	file1, err := os.Create("./mux.go")
	if err != nil {
		panic(err)
	}
	defer file1.Close()
	err = t1.Execute(file1, tmpl)
	if err != nil {
		panic(err)
	}

	t2 := template.Must(template.New("template").Parse(SessionTmpl))
	file2, err := os.Create("./session.go")
	if err != nil {
		panic(err)
	}
	defer file2.Close()
	err = t2.Execute(file2, tmpl)
	if err != nil {
		panic(err)
	}
}

type Template struct {
	Package     string
	Subject     string
	RecvMessage string
	SendMessage string
}
