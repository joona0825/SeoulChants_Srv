package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// 200 성공
func success(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusOK)

	response, _ := json.Marshal(Response{Result: true, Data: data, MinVersion: "26"})
	fmt.Fprint(w, string(response))
}

// 404 에러 핸들러
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "404 page not found")

	log.Println("404 page not found: access tried to " + r.RequestURI)
}

// 500 에러 핸들러
func internalErrorHandler(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusInternalServerError)

	response, _ := json.Marshal(Response{Result: false, Data: "서버 내부 오류가 발생했습니다.", MinVersion: "26"})
	fmt.Fprint(w, string(response))

	log.Println("internalError: " + message)
}
