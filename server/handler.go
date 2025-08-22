package server

import (
	"embed"
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

var mainViewTemp = Templates.Lookup("webcam.html")

func Main(writer http.ResponseWriter, request *http.Request) {

	err := mainViewTemp.Execute(writer, nil)
	if err != nil {
		log.Println("Error executing template:", err)
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
