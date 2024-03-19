package main

import (
	"os"
	"text/template"
)

func main() {
	tmpl := Template{
		Package:     "main",
		Subject:     "Topic",
		RecvMessage: "SubMsg",
		SendMessage: "PubMsg",
	}

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
