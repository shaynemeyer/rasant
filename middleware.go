package rasant

import (
	"net/http"
	"strconv"

	"github.com/justinas/nosurf"
)

func (ras *Rasant) SessionLoad(next http.Handler) http.Handler {
	ras.InfoLog.Println("SessionLoad called")
	return ras.Session.LoadAndSave(next)
}

func (ras *Rasant) NoSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next);
	secure, _ := strconv.ParseBool(ras.config.cookie.secure)

	csrfHandler.ExemptGlob("/api/*")

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path: "/",
		Secure: secure,
		SameSite: http.SameSiteStrictMode,
		Domain: ras.config.cookie.domain,
	})

	return csrfHandler
}