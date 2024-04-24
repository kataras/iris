package api

import (
	"time"

	"myapp/cache"
	"myapp/entity"
	"myapp/service"
	"myapp/sql"

	"github.com/kataras/iris/v12"
)

// ProductHandler is the http mux for products.
type ProductHandler struct {
	service *service.ProductService
	cache   *cache.Cache
}

// NewProductHandler returns the main controller for the products API.
func NewProductHandler(service *service.ProductService) *ProductHandler {
	return &ProductHandler{
		service: service,
		cache:   cache.New(service, "products", time.Minute),
	}
}

// GetByID fetches a single record from the database and sends it to the client.
// Method: GET.
func (h *ProductHandler) GetByID(ctx iris.Context) {
	id := ctx.Params().GetString("id")

	var product []byte
	err := h.cache.GetByID(ctx.Request().Context(), id, &product)
	if err != nil {
		if err == sql.ErrNoRows {
			writeEntityNotFound(ctx)
			return
		}

		debugf("ProductHandler.GetByID(id=%v): %v", id, err)
		writeInternalServerError(ctx)
		return
	}

	ctx.ContentType("application/json")
	ctx.Write(product)

	// ^ Could use our simple `noCache` or implement a Cache-Control (see kataras/iris/cache for that)
	// but let's keep it simple.
}

// List lists a set of records from the database.
// Method: GET.
func (h *ProductHandler) List(ctx iris.Context) {
	key := ctx.Request().URL.RawQuery

	products := []byte("[]")
	err := h.cache.List(ctx.Request().Context(), key, &products)
	if err != nil && err != sql.ErrNoRows {
		debugf("ProductHandler.List(DB) (%s): %v",
			key, err)

		writeInternalServerError(ctx)
		return
	}

	ctx.ContentType("application/json")
	ctx.Write(products)
}

// Create adds a record to the database.
// Method: POST.
func (h *ProductHandler) Create(ctx iris.Context) {
	var product entity.Product
	if err := ctx.ReadJSON(&product); err != nil {
		return
	}

	id, err := h.service.Insert(ctx.Request().Context(), product)
	if err != nil {
		if err == sql.ErrUnprocessable {
			ctx.StopWithJSON(iris.StatusUnprocessableEntity, newError(iris.StatusUnprocessableEntity, ctx.Request().Method, ctx.Path(), "required fields are missing"))
			return
		}

		debugf("ProductHandler.Create(DB): %v", err)
		writeInternalServerError(ctx)
		return
	}

	// Send 201 with body of {"id":$last_inserted_id"}.
	ctx.StatusCode(iris.StatusCreated)
	ctx.JSON(iris.Map{product.PrimaryKey(): id})
}

// Update performs a full-update of a record in the database.
// Method: PUT.
func (h *ProductHandler) Update(ctx iris.Context) {
	var product entity.Product
	if err := ctx.ReadJSON(&product); err != nil {
		return
	}

	affected, err := h.service.Update(ctx.Request().Context(), product)
	if err != nil {
		if err == sql.ErrUnprocessable {
			ctx.StopWithJSON(iris.StatusUnprocessableEntity, newError(iris.StatusUnprocessableEntity, ctx.Request().Method, ctx.Path(), "required fields are missing"))
			return
		}

		debugf("ProductHandler.Update(DB): %v", err)
		writeInternalServerError(ctx)
		return
	}

	status := iris.StatusOK
	if affected == 0 {
		status = iris.StatusNotModified
	}

	ctx.StatusCode(status)
}

// PartialUpdate is the handler for partially update one or more fields of the record.
// Method: PATCH.
func (h *ProductHandler) PartialUpdate(ctx iris.Context) {
	id := ctx.Params().GetInt64Default("id", 0)

	var attrs map[string]interface{}
	if err := ctx.ReadJSON(&attrs); err != nil {
		return
	}

	affected, err := h.service.PartialUpdate(ctx.Request().Context(), id, attrs)
	if err != nil {
		if err == sql.ErrUnprocessable {
			ctx.StopWithJSON(iris.StatusUnprocessableEntity, newError(iris.StatusUnprocessableEntity, ctx.Request().Method, ctx.Path(), "unsupported value(s)"))
			return
		}

		debugf("ProductHandler.PartialUpdate(DB): %v", err)
		writeInternalServerError(ctx)
		return
	}

	status := iris.StatusOK
	if affected == 0 {
		status = iris.StatusNotModified
	}

	ctx.StatusCode(status)
}

// Delete removes a record from the database.
// Method: DELETE.
func (h *ProductHandler) Delete(ctx iris.Context) {
	id := ctx.Params().GetInt64Default("id", 0)

	affected, err := h.service.DeleteByID(ctx.Request().Context(), id)
	if err != nil {
		debugf("ProductHandler.Delete(DB): %v", err)
		writeInternalServerError(ctx)
		return
	}

	status := iris.StatusOK // StatusNoContent
	if affected == 0 {
		status = iris.StatusNotModified
	}

	ctx.StatusCode(status)
}
