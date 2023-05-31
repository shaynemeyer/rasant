package render

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type Render struct {
	Renderer string
	RootPath string
	Secure bool
	Port string
	ServerName string
}

type TemplateData struct {
	IsAuthenticated bool
	IntMap map[string]int
	StringMap map[string]string
	FloatMap map[string]float32
	Data map[string]interface{}
	CSRFToken string
	Port string
	ServerName string
	Secure bool
}

func (ren *Render) Page(w http.ResponseWriter, r *http.Request, view string, variables, data interface{}) error {
	switch strings.ToLower(ren.Renderer) {
	case "go":
		return ren.GoPage(w, r, view, data)
	case "jet":
	}
	
	return nil
}

func (ren *Render) GoPage(w http.ResponseWriter, r *http.Request, view string, data interface{}) error {
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/views/%s.page.tmpl", ren.RootPath, view))
	if err!= nil {
    return err
  }

	td := &TemplateData{}
	if data != nil {
		td = data.(*TemplateData)
	}

	err = tmpl.Execute(w, &td)
	if err!= nil {
    return err
  }
	
	return nil
}

func (ren *Render) JetPage(w http.ResponseWriter, r *http.Request, view string, data interface{}) error {
	return nil
}