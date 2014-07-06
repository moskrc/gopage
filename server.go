package main

import (
	"./utils"
	"fmt"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/go-martini/martini"
	"html/template"
	"log"
	"net/http"
	"time"
)

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
			{"isset": utils.IsSet}, // example
		},
	}))

	m.Use(func(res http.ResponseWriter, req *http.Request) {
		data := &ContextProcessor{}
		data.CURRENT_PATH = req.URL.Path
		m.Map(data)
	})

	m.Get("/", func(r render.Render, ctx *ContextProcessor) {
		fmt.Println(utils.A)
		utils.A = 3
		fmt.Println(utils.A)
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

	utils.SendEmail("smtp.gmail.com", 587, nf.Email, "xxx", []string{"xxx@xxx.com"}, "Новый запрос", nf.Body)

	resp := map[string]interface{}{"ctx": ctx, "nf": nf, "success": true, "errors": errors}

	r.HTML(200, "feedback", resp)
}
