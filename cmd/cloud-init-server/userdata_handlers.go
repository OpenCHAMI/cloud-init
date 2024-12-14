package main

import "net/http"

func UserDataHandler(w http.ResponseWriter, r *http.Request) {
	payload := `#cloud-config`
	w.Write([]byte(payload))
}
