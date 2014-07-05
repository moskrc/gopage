package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/go-martini/martini"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"reflect"
	"runtime"
	"strings"
	"time"
)

func IsSet(a interface{}, key interface{}) bool {
	av := reflect.ValueOf(a)
	kv := reflect.ValueOf(key)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Slice:
		if int64(av.Len()) > kv.Int() {
			return true
		}
	case reflect.Map:
		if kv.Type() == av.Type().Key() {
			return av.MapIndex(kv).IsValid()
		}
	}

	return false
}

type Feedback struct {
	Created int64
	Body    string `form:"body" binding:"required"`
	Email   string `form:"email"`
}

type ContextProcessor struct {
	CURRENT_PATH string
}

func (f Feedback) Validate(errors *binding.Errors, req *http.Request) {
	if len(f.Body) > 0 && len(f.Body) < 10 {
		errors.Fields["Body"] = "Body is very short"
	}
}

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

func main() {
	m := martini.Classic()

	m.Use(martini.Logger)
	m.Use(martini.Static("static"))
	m.Use(render.Renderer(render.Options{
		Layout: "layout",

		Funcs: []template.FuncMap{
			{"isset": IsSet}, // example
		},
	}))

	m.Use(func(res http.ResponseWriter, req *http.Request) {
		data := &ContextProcessor{}
		data.CURRENT_PATH = req.URL.Path
		m.Map(data)
	})

	m.Get("/", func(r render.Render, ctx *ContextProcessor) {
		resp := map[string]interface{}{"ctx": ctx}
		r.HTML(200, "index", resp)
	})

	m.Get("/about/", func(r render.Render, ctx *ContextProcessor) {
		resp := map[string]interface{}{"ctx": ctx}
		r.HTML(200, "about", resp)
	})

	m.Group("/feedback", func(r martini.Router) {
		r.Get("/", binding.Form(Feedback{}), GetFeedback)
		r.Post("/", binding.Form(Feedback{}), SendFeedback)
	})

	m.Run()
}

func newFeedback(email, body string) Feedback {
	return Feedback{
		Created: time.Now().Unix(),
		Email:   email,
		Body:    body,
	}
}

func GetFeedback(ctx *ContextProcessor, f Feedback, m Feedback, c martini.Context, req *http.Request, r render.Render) {
	resp := map[string]interface{}{"ctx": ctx}

	r.HTML(200, "feedback", resp)
}

func SendFeedback(errors binding.Errors, ctx *ContextProcessor, f Feedback, c martini.Context, req *http.Request, r render.Render) {
	fmt.Printf("%+v", errors)
	nf := newFeedback(f.Email, f.Body)

	SendEmail(
		"smtp.gmail.com",
		587,
		nf.Email,
		"xxx",
		[]string{"xxx@xxx.com"},
		"Новый запрос",
		nf.Body)

	resp := map[string]interface{}{"ctx": ctx, "nf": nf, "success": true, "errors": errors}

	r.HTML(200, "feedback", resp)
}

func _CatchPanic(err *error, functionName string) {
	if r := recover(); r != nil {
		fmt.Printf("%s : PANIC Defered : %v\n", functionName, r)

		// Capture the stack trace
		buf := make([]byte, 10000)
		runtime.Stack(buf, false)

		fmt.Printf("%s : Stack Trace : %s", functionName, string(buf))

		if err != nil {
			*err = errors.New(fmt.Sprintf("%v", r))
		}
	} else if err != nil && *err != nil {
		fmt.Printf("%s : ERROR : %v\n", functionName, *err)

		// Capture the stack trace
		buf := make([]byte, 10000)
		runtime.Stack(buf, false)

		fmt.Printf("%s : Stack Trace : %s", functionName, string(buf))
	}
}

func SendEmail(host string, port int, userName string, password string,
	to []string, subject string, message string) (err error) {
	defer _CatchPanic(&err, "SendEmail")

	parameters := &struct {
		From    string
		To      string
		Subject string
		Message string
	}{
		userName,
		strings.Join([]string(to), ","),
		subject,
		message,
	}

	buffer := new(bytes.Buffer)

	template := template.Must(template.New("emailTemplate").Parse(_EmailScript()))
	template.Execute(buffer, parameters)

	auth := smtp.PlainAuth("", userName, password, host)

	err = smtp.SendMail(
		fmt.Sprintf("%s:%d", host, port),
		auth,
		userName,
		to,
		buffer.Bytes())

	return err
}

// _EmailScript returns a template for the email message to be sent
func _EmailScript() (script string) {
	return `From: {{.From}}
To: {{.To}}
Subject: {{.Subject}}
MIME-version: 1.0
Content-Type: text/html; charset="UTF-8"

{{.Message}}`
}
