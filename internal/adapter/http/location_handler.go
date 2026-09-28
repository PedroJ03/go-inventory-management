package httpadapter

import (
	"embed"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/PedroJ03/go-inventory-management/internal/application"
	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

//go:embed templates/location_form.html
var locationTemplateFiles embed.FS

type LocationHandler struct {
	locations  application.LocationRepository
	stockLevel application.StockLevelRepository
	template   *template.Template
}

type locationFormData struct {
	Name     string
	Street   string
	City     string
	Location *domain.Location
	Message  string
	Errors   []string
}

func NewLocationHandler(locations application.LocationRepository, stockLevel application.StockLevelRepository) (*LocationHandler, error) {
	if locations == nil {
		return nil, errors.New("location handler repository must not be nil")
	}
	if stockLevel == nil {
		return nil, errors.New("location handler stock level repository must not be nil")
	}
	parsed, err := template.ParseFS(locationTemplateFiles, "templates/location_form.html")
	if err != nil {
		return nil, errors.New("parse location template: " + err.Error())
	}
	return &LocationHandler{locations: locations, stockLevel: stockLevel, template: parsed}, nil
}

func (h *LocationHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	switch {
	case request.Method == http.MethodGet && request.URL.Path == "/locations/new":
		h.render(response, http.StatusOK, locationFormData{})
	case request.Method == http.MethodPost && request.URL.Path == "/locations":
		h.create(response, request)
	case request.Method == http.MethodPost:
		h.lifecycle(response, request)
	default:
		http.NotFound(response, request)
	}
}

func (h *LocationHandler) create(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		h.render(response, http.StatusBadRequest, locationFormData{Errors: []string{"The submitted form could not be read."}})
		return
	}
	data := locationFormData{
		Name:   request.FormValue("name"),
		Street: request.FormValue("street"),
		City:   request.FormValue("city"),
	}
	location, err := application.CreateLocation(request.Context(), h.locations, application.CreateLocationInput{
		Name: data.Name, Street: data.Street, City: data.City,
	})
	if err != nil {
		data.Errors = []string{locationErrorMessage(err)}
		h.render(response, http.StatusBadRequest, data)
		return
	}
	data.Location = &location
	data.Message = "Location created successfully."
	h.render(response, http.StatusCreated, data)
}

func (h *LocationHandler) lifecycle(response http.ResponseWriter, request *http.Request) {
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "locations" {
		http.NotFound(response, request)
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		h.render(response, http.StatusBadRequest, locationFormData{Errors: []string{"The location ID must be a positive number."}})
		return
	}

	var location domain.Location
	switch parts[2] {
	case "activate":
		location, err = application.ActivateLocation(request.Context(), h.locations, id)
	case "deactivate":
		location, err = application.DeactivateLocation(request.Context(), h.locations, h.stockLevel, id)
	default:
		http.NotFound(response, request)
		return
	}
	if err != nil {
		h.render(response, http.StatusBadRequest, locationFormData{Errors: []string{locationErrorMessage(err)}})
		return
	}

	h.render(response, http.StatusOK, locationFormData{Location: &location, Message: "Location status updated successfully."})
}

func (h *LocationHandler) render(response http.ResponseWriter, status int, data locationFormData) {
	response.WriteHeader(status)
	_ = h.template.Execute(response, data)
}

func locationErrorMessage(err error) string {
	switch {
	case errors.Is(err, application.ErrInvalidLocation):
		return "Enter a name, street, and city for the location."
	case errors.Is(err, application.ErrLocationHasStock):
		return "The location cannot be deactivated while it has positive stock."
	case errors.Is(err, application.ErrLocationNotFound):
		return "The location could not be found."
	default:
		return "The location action could not be completed. Please try again."
	}
}
