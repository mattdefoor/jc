package main

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGetHashNoParameter(t *testing.T) {
	req, err := http.NewRequest("GET", "/hash", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(hashHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestGetHashUnknownHash(t *testing.T) {
	req, err := http.NewRequest("GET", "/hash/999", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(hashHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestGetHashNotYetHashed(t *testing.T) {
	form := url.Values{}
	form.Add("password", "angryMonkey1")
	preq, err := http.NewRequest("POST", "/hash", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	preq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(hashHandler())
	handler.ServeHTTP(rr, preq)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("post handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "1"
	if rr.Body.String() != expected {
		t.Errorf("post handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	req, err := http.NewRequest("GET", "/hash/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(hashHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestGetHashAndWait(t *testing.T) {
	password := "angryMonkey2"
	form := url.Values{}
	form.Add("password", password)
	preq, err := http.NewRequest("POST", "/hash", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	preq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(hashHandler())
	handler.ServeHTTP(rr, preq)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("post handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "2"
	if rr.Body.String() != expected {
		t.Errorf("post handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	time.Sleep(time.Duration(hashWaitVar+1) * time.Second)
	req, err := http.NewRequest("GET", "/hash/2", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(hashHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("get handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	hash := fmt.Sprintf("%x", sha512.Sum512([]byte(password)))
	expected = base64.StdEncoding.EncodeToString([]byte(hash))
	if rr.Body.String() != expected {
		t.Errorf("get handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestStatsHandlerGet(t *testing.T) {
	req, err := http.NewRequest("GET", "/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(statsHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestStatsHandlerPost(t *testing.T) {
	req, err := http.NewRequest("POST", "/stats", strings.NewReader("z=data"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(statsHandler())

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}
