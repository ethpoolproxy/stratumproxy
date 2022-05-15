package webui

import (
	"github.com/Masterminds/sprig/v3"
	"html/template"
	"io"
)

type page struct {
	Pages  []string
	Data   interface{}
	Writer io.Writer
}

func (p *page) Build() error {
	var pages []string
	pages = append(pages, "template/root.gohtml")
	for _, s := range p.Pages {
		pages = append(pages, "template/"+s+".gohtml")
	}

	ts, err := template.New("root").Funcs(sprig.FuncMap()).Funcs(funcMap).ParseFS(pageTemplate, pages...)
	if err != nil {
		return err
	}

	err = ts.Execute(p.Writer, p.Data)
	if err != nil {
		return err
	}

	return nil
}
