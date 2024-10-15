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

func exec(req *http.Request) ([]byte, *http.Response, error) {
	var res *http.Response
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(consts.X_REQUEST_ID_LABEL, uuid.NewString())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, res, err
	}
	result, err := io.ReadAll(res.Body)
	return result, res, err
}

func login() (*http.Request, error) {
	return http.NewRequest(
		"PUT",
		fmt.Sprintf("%s%s/auth/login", os.Getenv("API_URL"), consts.BaseAPI_V1),
		strings.NewReader(fmt.Sprintf(`{"login": "%s","password":"%s"}`, os.Getenv("CLIENT_LOGIN"), os.Getenv("CLIENT_PASSWORD"))),
	)
}

func deactivate() (*http.Request, error) {
	req, err := login()
	if err != nil {
		return nil, err
	}
	_, res, err := exec(req)
	if err != nil {
		return nil, err
	}
	req, err = http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s%s/auth/deactivate", os.Getenv("API_URL"), consts.BaseAPI_V1),
		nil,
	)
	if err != nil {
		return nil, err
	}
	if len(res.Header["Set-Cookie"]) == 0 {
		req.AddCookie(&http.Cookie{
			Name:  "Authorization",
			Value: "Bearer " + os.Getenv("CLIENT_JWT"),
		})
		return req, nil
	}
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + strings.Split(res.Header["Set-Cookie"][0], " ")[1],
	})
	return req, nil
}

func (c apiCall) trigger() error {
	var req *http.Request
	var err error
	switch c.service {
	case "auth":
		switch c.action {
		case "login":
			req, err = login()
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
		case "deactivate":
			req, err = deactivate()
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
				Value: "Bearer " + os.Getenv("CLIENT_JWT"),
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
				Value: "Bearer " + os.Getenv("CLIENT_JWT"),
			})
		}

	}
	result, res, err := exec(req)
	if err != nil {
		return err
	}
	fmt.Printf("Response: %d\n%s\n Headers: %+v\n", res.StatusCode, string(result), res.Header)
	return nil
}

func newApiCall(service, action string) call {
	return apiCall{service, action}
}
