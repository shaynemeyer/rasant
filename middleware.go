package rasant

import "net/http"

func (ras *Rasant) SessionLoad(next http.Handler) http.Handler {
	ras.InfoLog.Println("SessionLoad called")
	return ras.Session.LoadAndSave(next)
}
