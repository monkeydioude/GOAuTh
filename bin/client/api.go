package main

import (
	"GOAuTh/internal/config/consts"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

type apiCall struct {
	service string
	action  string
}

func (c apiCall) trigger() error {
	var res *http.Response
	var req *http.Request
	var err error
	switch c.service {
	case "auth":
		switch c.action {
		case "login":
			req, err = http.NewRequest(
				"PUT",
				fmt.Sprintf("%s%s/auth/login", os.Getenv("API_URL"), consts.BaseAPI_V1),
				strings.NewReader(fmt.Sprintf(`{"login": "%s","password":"%s"}`, os.Getenv("CLIENT_LOGIN"), os.Getenv("CLIENT_PASSWORD"))),
			)
			if err != nil {
				return err
			}
		case "signup":
			req, err = http.NewRequest(
				"POST",
				fmt.Sprintf("%s%s/auth/signup", os.Getenv("API_URL"), consts.BaseAPI_V1),
				strings.NewReader(fmt.Sprintf(`{"login": "%s","password":"%s"}`, os.Getenv("CLIENT_LOGIN"), os.Getenv("CLIENT_PASSWORD"))),
			)
			if err != nil {
				return err
			}
		}
	case "jwt":
		switch c.action {
		case "status":
			req, err = http.NewRequest(
				"GET",
				fmt.Sprintf("%s%s/jwt/status", os.Getenv("API_URL"), consts.BaseAPI_V1),
				nil,
			)
			if err != nil {
				return err
			}
			req.AddCookie(&http.Cookie{
				Name:  "Authorization",
				Value: os.Getenv("CLIENT_JWT"),
			})
		case "refresh":
			req, err = http.NewRequest(
				"PUT",
				fmt.Sprintf("%s%s/jwt/refresh", os.Getenv("API_URL"), consts.BaseAPI_V1),
				nil,
			)
			if err != nil {
				return err
			}
			req.AddCookie(&http.Cookie{
				Name:  "Authorization",
				Value: os.Getenv("CLIENT_JWT"),
			})
		}

	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(consts.X_REQUEST_ID_LABEL, uuid.NewString())
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	result, err := io.ReadAll(res.Body)
	fmt.Printf("Response: %d\n%s\n", res.StatusCode, string(result))
	return err
}

func newApiCall(service, action string) call {
	return apiCall{service, action}
}
