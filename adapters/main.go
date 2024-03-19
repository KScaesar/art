package main

import (
	"os"
	"text/template"
)

func main() {
	tmpl := Template{
		Package:     "redis",
		Subject:     "Channel",
		RecvMessage: "RedisMessage",
		SendMessage: "Event",
	}

	t1 := template.Must(template.New("template").Parse(MuxTmpl))
	file1, err := os.Create("./redis/mux.go")
	if err != nil {
		panic(err)
	}
	defer file1.Close()
	err = t1.Execute(file1, tmpl)
	if err != nil {
		panic(err)
	}

	t2 := template.Must(template.New("template").Parse(SessionTmpl))
	file2, err := os.Create("./redis/session.go")
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
