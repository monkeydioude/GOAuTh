package auth

import (
	"GOAuTh/internal/entities"
	"GOAuTh/internal/handlers"
	"GOAuTh/pkg/http/request"
	"GOAuTh/pkg/http/response"
	"log"
	"net/http"
)

// Signup would be the route user when creating a new user
func Signup(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	rawPayload := request.Json[entities.DefaultUser](req)
	if rawPayload.IsErr() {
		response.InternalServerError(rawPayload.Error.Error(), w)
		return
	}

	payload := rawPayload.Result()
	h.HydrateEntity(payload)
	if err := h.LoginConstraint(payload.Login); err != nil {
		response.UnprocessableEntity(err.Error(), w)
		return
	}

	if res := h.DB.Save(payload); res.Error != nil {
		log.Printf("[ERR ] %s\n", res.Error.Error())
		response.InternalServerError("Could not save to DB. Possible duplicate entry", w)
		return
	}
	// LogIn(h, w, req)
	response.JsonOk(w)
}
