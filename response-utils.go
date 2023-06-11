package rasant

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
)

func (ras *Rasant) WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	if len(headers) > 0 { 
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func (ras *Rasant) WriteXML(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error { 
	out, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if len(headers) > 0 { 
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)

	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func (ras *Rasant) DownloadFile(w http.ResponseWriter, r *http.Request, pathToFile, fileName string) error {	
	fp := path.Join(pathToFile, fileName)
	fileToServe := filepath.Clean(fp)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; file=\"%s\"", fileName))
	http.ServeFile(w, r, fileToServe)
	
	return nil
}

func (ras *Rasant) Error404(w http.ResponseWriter, r *http.Request) {
	ras.ErrorStatus(w, http.StatusNotFound)
}

func (ras *Rasant) Error500(w http.ResponseWriter, r *http.Request) {
	ras.ErrorStatus(w, http.StatusInternalServerError)
}

func (ras *Rasant) ErrorUnauthorized(w http.ResponseWriter, r *http.Request) {
	ras.ErrorStatus(w, http.StatusUnauthorized)
}

func (ras *Rasant) ErrorForbidden(w http.ResponseWriter, r *http.Request) {
	ras.ErrorStatus(w, http.StatusForbidden)
}

func (ras *Rasant) ErrorStatus(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

