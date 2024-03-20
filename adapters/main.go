package main

import (
	"flag"
	"os"
	"text/template"
)

func main() {
	tmpl := Template{}
	ok := LoadDataFromCli(&tmpl)
	if ok {
		WriteTemplate(tmpl)
	}
}

func LoadDataFromCli(tmpl *Template) bool {
	var dir string
	var pkg string
	var subject string
	var recv string
	var send string

	flag.StringVar(&dir, "dir", "./", "generate code to dir")
	flag.StringVar(&pkg, "pkg", "main", "Package")
	flag.StringVar(&subject, "s", "Channel", "Subject")
	flag.StringVar(&recv, "recv", "ConsumeMsg", "RecvMessage Name")
	flag.StringVar(&send, "send", "ProduceMsg", "SendMessage Name")

	help := flag.Bool("h", false, "help")
	flag.Parse()

	if *help {
		os.Stdout.WriteString(`help: 
    aritfex -dir ./ -pkg kafka -s Topic -recv SubMsg -send PubMsg
`)
		return false
	}

	if dir[len(dir)-1] != '/' {
		dir += "/"
	}

	tmpl.Dir = dir
	tmpl.Package = pkg
	tmpl.Subject = subject
	tmpl.RecvMessage = recv
	tmpl.SendMessage = send
	return true
}

func WriteTemplate(tmpl Template) {
	t1 := template.Must(template.New("template").Parse(MsgTmpl))
	file1, err := os.Create(tmpl.Dir + "message.go")
	if err != nil {
		panic(err)
	}
	defer file1.Close()
	err = t1.Execute(file1, tmpl)
	if err != nil {
		panic(err)
	}

	t2 := template.Must(template.New("template").Parse(SessionTmpl))
	file2, err := os.Create(tmpl.Dir + "session.go")
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
	Dir         string
	Package     string
	Subject     string
	RecvMessage string
	SendMessage string
}
