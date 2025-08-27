package main

import (
	"html/template"
	"log"
	"net/http"
)

type PageData struct {
	CSPNonce string
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	ts, err := template.ParseFiles("./ui/html/pages/home.tmpl.html")
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	/*
		nonce, ok := r.Context().Value(cspNonceKey{}).(string)
		if !ok {
			log.Print(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		data := PageData{
			CSPNonce: nonce,
		}

		if err := ts.Execute(w, data); err != nil {
	*/
	if err := ts.Execute(w, nil); err != nil {
		log.Print(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (app *application) search(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	defer conn.Close()
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = conn.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func (app *application) teapot(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	w.Write([]byte("i'm a teapot..."))
}
