package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/monkeydioude/goauth/internal/config/consts"

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
		strings.NewReader(fmt.Sprintf(`{"login": "%s","password":"%s", "realm_name":"%s"}`, os.Getenv("CLIENT_LOGIN"), os.Getenv("CLIENT_PASSWORD"), os.Getenv("CLIENT_REALM"))),
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
		fmt.Sprintf("%s%s/user/deactivate", os.Getenv("API_URL"), consts.BaseAPI_V1),
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
	cookies, err := http.ParseCookie(res.Header["Set-Cookie"][0])
	if err != nil {
		return nil, err
	}
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: cookies[0].Value,
	})
	return req, nil
}

func userPassword() (*http.Request, error) {
	os.Setenv("CLIENT_PASSWORD", os.Getenv("OLD_PASSWORD"))
	req, err := login()
	if err != nil {
		return nil, err
	}
	_, res, err := exec(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, errors.New("login did not give 200")
	}
	req, err = http.NewRequest(
		"PUT",
		fmt.Sprintf("%s%s/user/password", os.Getenv("API_URL"), consts.BaseAPI_V1),
		strings.NewReader(fmt.Sprintf(`{"password":"%s", "new_password": "%s"}`, os.Getenv("OLD_PASSWORD"), os.Getenv("NEW_PASSWORD"))),
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
	cookies, err := http.ParseCookie(res.Header["Set-Cookie"][0])
	if err != nil {
		return nil, err
	}
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: cookies[0].Value,
	})
	return req, nil
}

func userLogin() (*http.Request, error) {
	os.Setenv("CLIENT_LOGIN", os.Getenv("OLD_LOGIN"))
	req, err := login()
	if err != nil {
		return nil, err
	}
	_, res, err := exec(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, errors.New("login did not give 200")
	}
	req, err = http.NewRequest(
		"PUT",
		fmt.Sprintf("%s%s/user/login", os.Getenv("API_URL"), consts.BaseAPI_V1),
		strings.NewReader(fmt.Sprintf(`{"password":"%s", "login": "%s", "new_login": "%s"}`, os.Getenv("CLIENT_PASSWORD"), os.Getenv("OLD_LOGIN"), os.Getenv("NEW_LOGIN"))),
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
	cookies, err := http.ParseCookie(res.Header["Set-Cookie"][0])
	if err != nil {
		return nil, err
	}
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: cookies[0].Value,
	})
	return req, nil
}

func (c apiCall) trigger() error {
	var req *http.Request
	var err error

	switch c.service {
	case "realm":
		switch c.action {
		case "create":
			return realmCreate()
		case "view":
			return realmsShow()
		}
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
				strings.NewReader(fmt.Sprintf(`{"login": "%s","password":"%s", "realm_name": "%s"}`, os.Getenv("CLIENT_LOGIN"), os.Getenv("CLIENT_PASSWORD"), os.Getenv("CLIENT_REALM"))),
			)
			if err != nil {
				return err
			}
		}
	case "user":
		switch c.action {
		case "password":
			req, err = userPassword()
			if err != nil {
				return err
			}
		case "login":
			req, err = userLogin()
			if err != nil {
				return err
			}
		case "deactivate":
			req, err = deactivate()
			if err != nil {
				return err
			}
		case "change_user":
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
	default:
		return errors.New("unavailable through api yet")
	}
	result, res, err := exec(req)
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Response: %d\n%s\n Headers: %+v\n", res.StatusCode, string(result), res.Header))
	return nil
}

func newApiCall(service, action string) call {
	return apiCall{service, action}
}
