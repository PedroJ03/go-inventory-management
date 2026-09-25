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

//go:embed templates/sell_stock_form.html
var sellStockTemplateFiles embed.FS

type SellStockHandler struct {
	products  application.ProductRepository
	locations application.LocationRepository
	selling   application.StockSellingPort
	template  *template.Template
}

type sellStockFormData struct {
	ProductID  string
	LocationID string
	Units      string
	Result     *sellStockResultData
	Message    string
	Errors     []string
}

type sellStockResultData struct {
	Units      int64
	MovementID int64
}

func NewSellStockHandler(products application.ProductRepository, locations application.LocationRepository, selling application.StockSellingPort) (*SellStockHandler, error) {
	if products == nil {
		return nil, errors.New("sell stock handler product repository must not be nil")
	}
	if locations == nil {
		return nil, errors.New("sell stock handler location repository must not be nil")
	}
	if selling == nil {
		return nil, errors.New("sell stock handler selling port must not be nil")
	}
	parsed, err := template.ParseFS(sellStockTemplateFiles, "templates/sell_stock_form.html")
	if err != nil {
		return nil, errors.New("parse sell stock template: " + err.Error())
	}
	return &SellStockHandler{products: products, locations: locations, selling: selling, template: parsed}, nil
}

func (h *SellStockHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/stock/sell" {
		http.NotFound(response, request)
		return
	}
	switch request.Method {
	case http.MethodGet:
		h.render(response, http.StatusOK, sellStockFormData{})
	case http.MethodPost:
		h.sell(response, request)
	default:
		http.NotFound(response, request)
	}
}

func (h *SellStockHandler) sell(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		h.render(response, http.StatusBadRequest, sellStockFormData{Errors: []string{"The submitted form could not be read."}})
		return
	}

	data := sellStockFormData{
		ProductID:  strings.TrimSpace(request.FormValue("product_id")),
		LocationID: strings.TrimSpace(request.FormValue("location_id")),
		Units:      strings.TrimSpace(request.FormValue("units")),
	}
	input, err := parseSellStockInput(data)
	if err != nil {
		data.Errors = []string{err.Error()}
		h.render(response, http.StatusBadRequest, data)
		return
	}

	result, err := application.SellStock(request.Context(), h.products, h.locations, h.selling, input)
	if err != nil {
		status, message := sellStockError(err)
		data.Errors = []string{message}
		h.render(response, status, data)
		return
	}

	data.Message = "Stock sold successfully."
	data.Result = &sellStockResultData{Units: result.StockLevel.Units, MovementID: result.Movement.ID}
	h.render(response, http.StatusCreated, data)
}

func parseSellStockInput(data sellStockFormData) (application.SellStockInput, error) {
	productID, err := uuid.Parse(data.ProductID)
	if err != nil || productID == uuid.Nil {
		return application.SellStockInput{}, errors.New("Enter a valid product UUID.")
	}
	locationID, err := strconv.ParseInt(data.LocationID, 10, 64)
	if err != nil || locationID <= 0 {
		return application.SellStockInput{}, errors.New("Location ID must be a positive whole number.")
	}
	units, err := strconv.ParseInt(data.Units, 10, 64)
	if err != nil || units <= 0 {
		return application.SellStockInput{}, errors.New("Units must be a positive whole number.")
	}
	return application.SellStockInput{ProductID: productID, LocationID: locationID, Units: units}, nil
}

func (h *SellStockHandler) render(response http.ResponseWriter, status int, data sellStockFormData) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.WriteHeader(status)
	_ = h.template.Execute(response, data)
}

func sellStockError(err error) (int, string) {
	switch {
	case errors.Is(err, application.ErrProductNotFound), errors.Is(err, application.ErrLocationNotFound):
		return http.StatusNotFound, "The product or location could not be found."
	case errors.Is(err, application.ErrInactiveLocation):
		return http.StatusBadRequest, "The selected location is inactive and cannot sell stock."
	case errors.Is(err, application.ErrInsufficientStock):
		return http.StatusConflict, "There is not enough stock available at the selected location."
	case errors.Is(err, application.ErrInvalidSellingInput):
		return http.StatusBadRequest, "The selling details are invalid."
	default:
		return http.StatusInternalServerError, "The stock sale could not be completed. Please try again."
	}
}
