package httpadapter

import (
	"embed"
	"errors"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/pedroj/go-inventory-management/internal/application"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

//go:embed templates/product_form.html
var templateFiles embed.FS

var productUnits = []string{
	string(domain.UnitEach),
	string(domain.UnitMilliliters),
	string(domain.UnitLiters),
	string(domain.UnitGrams),
	string(domain.UnitKilograms),
}

type ProductHandler struct {
	repository application.ProductRepository
	template   *template.Template
}

type productFormData struct {
	Name    string
	Brand   string
	Amount  string
	Unit    string
	Units   []string
	Errors  []string
	Created *domain.Product
}

func NewProductHandler(repository application.ProductRepository) (*ProductHandler, error) {
	if repository == nil {
		return nil, errors.New("product handler repository must not be nil")
	}
	parsed, err := template.ParseFS(templateFiles, "templates/product_form.html")
	if err != nil {
		return nil, errors.New("parse product template: " + err.Error())
	}
	return &ProductHandler{repository: repository, template: parsed}, nil
}

func (h *ProductHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	switch {
	case request.Method == http.MethodGet && request.URL.Path == "/products/new":
		h.render(response, http.StatusOK, productFormData{Units: productUnits})
	case request.Method == http.MethodPost && request.URL.Path == "/products":
		h.create(response, request)
	default:
		http.NotFound(response, request)
	}
}

func (h *ProductHandler) create(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		h.render(response, http.StatusBadRequest, productFormData{Errors: []string{"The submitted form could not be read."}, Units: productUnits})
		return
	}
	data := productFormData{
		Name:   request.FormValue("name"),
		Brand:  request.FormValue("brand"),
		Amount: request.FormValue("amount"),
		Unit:   request.FormValue("unit"),
		Units:  productUnits,
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(data.Amount), 64)
	if err != nil || math.IsNaN(amount) || math.IsInf(amount, 0) {
		data.Errors = append(data.Errors, "Amount must be a valid number greater than zero.")
	}
	unit := domain.Unit(data.Unit)
	if !isProductUnit(unit) {
		data.Errors = append(data.Errors, "Select a supported unit.")
	}
	if len(data.Errors) > 0 {
		h.render(response, http.StatusBadRequest, data)
		return
	}

	product, err := application.CreateProduct(request.Context(), h.repository, application.CreateProductInput{
		Name: data.Name, Brand: data.Brand, Amount: amount, Unit: unit,
	})
	if err != nil {
		data.Errors = []string{productErrorMessage(err)}
		h.render(response, http.StatusBadRequest, data)
		return
	}
	data.Created = &product
	h.render(response, http.StatusCreated, data)
}

func (h *ProductHandler) render(response http.ResponseWriter, status int, data productFormData) {
	response.WriteHeader(status)
	if err := h.template.Execute(response, data); err != nil {
		return
	}
}

func isProductUnit(unit domain.Unit) bool {
	for _, supported := range productUnits {
		if string(unit) == supported {
			return true
		}
	}
	return false
}

func productErrorMessage(err error) string {
	switch {
	case errors.Is(err, application.ErrInvalidProduct):
		return "Check the product details and try again."
	case errors.Is(err, application.ErrProductAlreadyExists):
		return "A product with these details already exists."
	default:
		return "The product could not be created. Please try again."
	}
}
