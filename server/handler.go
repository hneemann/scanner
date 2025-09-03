package server

import (
	"embed"
	"github.com/hneemann/session"
	"html/template"
	"io"
	"log"
	"net/http"
	"scan/data"
	"strconv"
)

//go:embed assets/*
var Assets embed.FS

//go:embed templates/*
var templateFS embed.FS

var Templates = template.Must(template.New("").ParseFS(templateFS, "templates/*.html"))

var (
	mainViewTemp      = Templates.Lookup("webcam.html")
	documentaViewTemp = Templates.Lookup("documents.html")
)

func Main(writer http.ResponseWriter, request *http.Request) {
	if userData, ok := request.Context().Value("data").(*data.UserData); ok {
		err := mainViewTemp.Execute(writer, userData)
		if err != nil {
			log.Println("Error executing template:", err)
		}
	} else {
		http.Error(writer, "No data found in context", http.StatusInternalServerError)
		log.Println("No data found in context")
		return
	}
}

func Store(writer http.ResponseWriter, request *http.Request) {
	if userData, ok := request.Context().Value("data").(*data.UserData); ok {

		if request.Method == http.MethodPost {
			err := request.ParseMultipartForm(10 * 1024 * 1024)
			if err != nil {
				log.Println("Error parsing form:", err)
				http.Error(writer, "Error parsing form: "+err.Error(), http.StatusBadRequest)
				return
			}

			file, _, err := request.FormFile("scan")
			if err != nil {
				log.Println("Error retrieving file:", err)
				http.Error(writer, "No data provided", http.StatusBadRequest)
				return
			}

			data, err := io.ReadAll(file)

			err = userData.Add(data)
			if err != nil {
				log.Println("Error saving data:", err)
				http.Error(writer, "Error saving data: "+err.Error(), http.StatusInternalServerError)
				return
			}

			log.Println("Data received, size:", len(data), "bytes")

			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte("Data received successfully, size: " + strconv.Itoa(len(data)) + " bytes"))
		} else {
			http.Error(writer, "Operation not allowed", http.StatusInternalServerError)
			log.Println("Operation not allowed")
			return
		}
	} else {
		http.Error(writer, "No data found in context", http.StatusInternalServerError)
		log.Println("No data found in context")
		return
	}
}

func Create(writer http.ResponseWriter, request *http.Request) {
	if userData, ok := request.Context().Value("data").(*data.UserData); ok {

		err := userData.Create()
		if err != nil {
			log.Println("Error saving data:", err)
			http.Error(writer, "Error processing data: "+err.Error(), http.StatusInternalServerError)
			return
		}
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("Data processed successfully"))

	} else {
		http.Error(writer, "No data found in context", http.StatusInternalServerError)
		log.Println("No data found in context")
		return
	}
}

func Documents(writer http.ResponseWriter, request *http.Request) {
	if userData, ok := request.Context().Value("data").(*data.UserData); ok && userData.Download() {

		query := request.URL.Query()
		delStr := query.Get("delete")
		if delStr != "" {
			if delStr == "all" {
				err := userData.DeleteAll()
				if err != nil {
					log.Println("Error deleting all files:", err)
					http.Error(writer, "Error deleting all files: "+err.Error(), http.StatusInternalServerError)
					return
				}
				http.Redirect(writer, request, "/documents/", http.StatusSeeOther)
				return
			} else {
				index, err := strconv.Atoi(delStr)
				if err == nil && index >= 0 && index < len(userData.Documents) {
					err = userData.Delete(index)
					if err != nil {
						log.Println("Error deleting file:", err)
						http.Error(writer, "Error deleting file: "+err.Error(), http.StatusInternalServerError)
						return
					}
					http.Redirect(writer, request, "/documents/", http.StatusSeeOther)
					return
				}
			}
		}
		downloadStr := query.Get("download")
		if downloadStr != "" {
			index, err := strconv.Atoi(downloadStr)
			if err == nil && index >= 0 && index < len(userData.Documents) {
				doc := userData.Documents[index]
				r, err := userData.Reader(doc.Name)
				if err != nil {
					log.Println("Error reading file:", err)
					http.Error(writer, "Error reading file: "+err.Error(), http.StatusInternalServerError)
					return
				}
				defer func() {
					err := r.Close()
					if err != nil {
						log.Println("Error closing file:", err)
					}
				}()
				var contentType string
				if doc.Type == data.TypePDF {
					contentType = "application/pdf"
				} else if doc.Type == data.TypeJPEG {
					contentType = "image/jpeg"
				} else {
					contentType = "application/octet-stream"
				}
				writer.Header().Set("Content-Type", contentType)
				writer.Header().Set("Content-Disposition", "attachment; filename=\""+doc.Name+"\"")
				writer.WriteHeader(http.StatusOK)
				_, err = io.Copy(writer, r)
				if err != nil {
					log.Println("Error writing file to response:", err)
				} else {
					doc.Processed = true
					http.Redirect(writer, request, "/documents/", http.StatusSeeOther)
				}
				return
			}
		}

		err := documentaViewTemp.Execute(writer, userData.Documents)
		if err != nil {
			log.Println("Error executing template:", err)
		}

	} else {
		http.Error(writer, "No data found in context", http.StatusInternalServerError)
		log.Println("No data found in context")
		return
	}
}

func LoginAnonymous(sm *session.Cache[data.UserData]) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		id, err := sm.CreateSessionToken("anonymous", "")
		if err == nil {
			http.SetCookie(writer, session.CreateSecureCookie("id", id))
		}
		ud := sm.GetSessionData(id)
		if ud != nil {
			ud.NoDownload()
		}
		http.Redirect(writer, request, "/", http.StatusSeeOther)
	}
}
