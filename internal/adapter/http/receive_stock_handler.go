package httpadapter

import (
	"embed"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/pedroj/go-inventory-management/internal/application"
)

//go:embed templates/receive_stock_form.html
var receiveStockTemplateFiles embed.FS

type ReceiveStockHandler struct {
	products  application.ProductRepository
	locations application.LocationRepository
	receiving application.StockReceivingPort
	template  *template.Template
}

type receiveStockFormData struct {
	ProductID  string
	LocationID string
	Units      string
	Result     *receiveStockResultData
	Message    string
	Errors     []string
}

type receiveStockResultData struct {
	Units      int64
	MovementID int64
}

func NewReceiveStockHandler(products application.ProductRepository, locations application.LocationRepository, receiving application.StockReceivingPort) (*ReceiveStockHandler, error) {
	if products == nil {
		return nil, errors.New("receive stock handler product repository must not be nil")
	}
	if locations == nil {
		return nil, errors.New("receive stock handler location repository must not be nil")
	}
	if receiving == nil {
		return nil, errors.New("receive stock handler receiving port must not be nil")
	}
	parsed, err := template.ParseFS(receiveStockTemplateFiles, "templates/receive_stock_form.html")
	if err != nil {
		return nil, errors.New("parse receive stock template: " + err.Error())
	}
	return &ReceiveStockHandler{products: products, locations: locations, receiving: receiving, template: parsed}, nil
}

func (h *ReceiveStockHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/stock/receive" {
		http.NotFound(response, request)
		return
	}
	switch request.Method {
	case http.MethodGet:
		h.render(response, http.StatusOK, receiveStockFormData{})
	case http.MethodPost:
		h.receive(response, request)
	default:
		http.NotFound(response, request)
	}
}

func (h *ReceiveStockHandler) receive(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		h.render(response, http.StatusBadRequest, receiveStockFormData{Errors: []string{"The submitted form could not be read."}})
		return
	}

	data := receiveStockFormData{
		ProductID:  strings.TrimSpace(request.FormValue("product_id")),
		LocationID: strings.TrimSpace(request.FormValue("location_id")),
		Units:      strings.TrimSpace(request.FormValue("units")),
	}
	input, err := parseReceiveStockInput(data)
	if err != nil {
		data.Errors = []string{err.Error()}
		h.render(response, http.StatusBadRequest, data)
		return
	}

	result, err := application.ReceiveStock(request.Context(), h.products, h.locations, h.receiving, input)
	if err != nil {
		status, message := receiveStockError(err)
		data.Errors = []string{message}
		h.render(response, status, data)
		return
	}

	data.Message = "Stock received successfully."
	data.Result = &receiveStockResultData{Units: result.StockLevel.Units, MovementID: result.Movement.ID}
	h.render(response, http.StatusCreated, data)
}

func parseReceiveStockInput(data receiveStockFormData) (application.ReceiveStockInput, error) {
	productID, err := uuid.Parse(data.ProductID)
	if err != nil || productID == uuid.Nil {
		return application.ReceiveStockInput{}, errors.New("Enter a valid product UUID.")
	}
	locationID, err := strconv.ParseInt(data.LocationID, 10, 64)
	if err != nil || locationID <= 0 {
		return application.ReceiveStockInput{}, errors.New("Location ID must be a positive whole number.")
	}
	units, err := strconv.ParseInt(data.Units, 10, 64)
	if err != nil || units <= 0 {
		return application.ReceiveStockInput{}, errors.New("Units must be a positive whole number.")
	}
	return application.ReceiveStockInput{ProductID: productID, LocationID: locationID, Units: units}, nil
}

func (h *ReceiveStockHandler) render(response http.ResponseWriter, status int, data receiveStockFormData) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.WriteHeader(status)
	_ = h.template.Execute(response, data)
}

func receiveStockError(err error) (int, string) {
	switch {
	case errors.Is(err, application.ErrProductNotFound), errors.Is(err, application.ErrLocationNotFound):
		return http.StatusNotFound, "The product or location could not be found."
	case errors.Is(err, application.ErrInactiveLocation):
		return http.StatusBadRequest, "The selected location is inactive and cannot receive stock."
	case errors.Is(err, application.ErrInvalidReceivingInput):
		return http.StatusBadRequest, "The receiving details are invalid."
	default:
		return http.StatusInternalServerError, "The stock receipt could not be completed. Please try again."
	}
}
