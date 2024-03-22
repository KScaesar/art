package main

import (
	"flag"
	"os"
	"text/template"
)

func main() {
	tmpl := Template{}
	ok := LoadDataFromCli(&tmpl)
	if !ok {
		return
	}
	OpenFileAndRenderTemplate(tmpl, "message", MsgTmpl)
	OpenFileAndRenderTemplate(tmpl, "session", SessionTmpl)
}

func PrintHelp(detail bool) {
	const example = `help: 
    artifex gen -dir ./ -pkg infra -f kafka -s Topic -recv SubMsg -send PubMsg
`
	const text = `
-dir  Generate code to dir
-f    File prefix name
-pkg  Package name
-s    Subject name
-recv RecvMessage name
-send SendMessage name
`

	if !detail {
		os.Stdout.WriteString(example)
		return
	}

	os.Stdout.WriteString(example + text)
}

func LoadDataFromCli(tmpl *Template) bool {
	cmdHelp := flag.NewFlagSet("help", flag.ExitOnError)
	cmdGen := flag.NewFlagSet("gen", flag.ExitOnError)

	args := os.Args
	if len(args) < 2 {
		PrintHelp(false)
		return false
	}

	switch os.Args[1] {
	case "gen":
		var dir string
		var pkg string
		var file string
		var subject string
		var recv string
		var send string

		cmdGen.StringVar(&dir, "dir", "./", "Generate code to dir")
		cmdGen.StringVar(&pkg, "pkg", "main", "Package name")
		cmdGen.StringVar(&file, "f", "", "File prefix name")
		cmdGen.StringVar(&subject, "s", "Channel", "Subject name")
		cmdGen.StringVar(&recv, "recv", "ConsumeMsg", "RecvMessage name")
		cmdGen.StringVar(&send, "send", "ProduceMsg", "SendMessage name")
		help := cmdGen.Bool("h", false, "Help")

		if cmdGen.Parse(os.Args[2:]) != nil {
			return false
		}

		if *help {
			PrintHelp(true)
			return false
		}

		tmpl.FileDir = dir
		tmpl.FileName = file
		tmpl.Package = pkg
		tmpl.Subject = subject
		tmpl.RecvMessage = recv
		tmpl.SendMessage = send
		return true

	default:
		help := cmdHelp.Bool("h", false, "Help")
		cmdHelp.Parse(os.Args[1:])
		if *help {
			PrintHelp(true)
			return false
		}
		PrintHelp(false)
		return false
	}
}

func OpenFileAndRenderTemplate(tmpl Template, postfix string, text string) (err error) {
	defer func() {
		if err != nil {
			panic(err)
		}
	}()

	err = os.MkdirAll(tmpl.Dir(), 0755)
	if err != nil {
		return err
	}

	path := tmpl.Path(postfix)

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		file, err = os.Create(path)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	t := template.Must(template.New(path).Parse(text))
	return t.Execute(file, tmpl)
}

type Template struct {
	FileDir     string
	FileName    string
	Package     string
	Subject     string
	RecvMessage string
	SendMessage string
}

func (t *Template) Path(postfix string) string {
	dir := t.Dir()
	name := t.FileName
	if name != "" {
		name += "_"
	}
	return dir + name + postfix + ".go"
}

func (t *Template) Dir() string {
	dir := t.FileDir
	if dir[len(dir)-1] != '/' {
		dir += "/"
	}
	return dir
}
