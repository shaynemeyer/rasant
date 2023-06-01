package rasant

import "net/http"

func (ras *Rasant) SessionLoad(next http.Handler) http.Handler {
	ras.InfoLog.Panicln("SessionLoad called")
	return ras.Session.LoadAndSave(next)
}
