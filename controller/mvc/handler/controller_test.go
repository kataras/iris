package handler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	irisContext "github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"

	"uuid"
)

type (
	Recipe struct {
		ID    string       `json:"id"`
		Name  string       `json:"name"`
		Items []RecipeItem `json:"items"`
	}

	// See controller_param_test.go for parameter types test.
	//
	// RecipeParam struct {
	// 	RecipeID string `param:"recipe_id"`
	// }

	RecipeFilter struct {
		Name string `json:"name"`
	}

	RecipeItem struct {
		ID       string  `json:"id"`
		FoodID   string  `json:"food_id"`
		Quantity float64 `json:"quantity"` // specific quantity in grams for this recipe's item.
	}

	// RecipeItemParam struct {
	// 	RecipeParam
	// 	ItemID string `param:"item_id"`
	// }
)

type Repository[T, V any] interface {
	Create(ctx context.Context, obj T) (string, error)
	Update(ctx context.Context, obj T) (bool, error)
	Delete(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (T, error)
	List(ctx context.Context, filter V, limit, offset int) ([]T, int, error)
}

type FoodRepository struct {
	foods map[string]Food
}

var _ Repository[Food, FoodFilter] = (*FoodRepository)(nil)

func NewFoodRepository() *FoodRepository {
	return &FoodRepository{
		foods: make(map[string]Food),
	}
}

func (r *FoodRepository) Create(ctx context.Context, food Food) (string, error) {
	food.ID = uuid.New().String()
	r.foods[food.ID] = food
	return food.ID, nil
}

func (r *FoodRepository) Update(ctx context.Context, food Food) (bool, error) {
	if _, ok := r.foods[food.ID]; !ok {
		return false, nil
	}

	r.foods[food.ID] = food
	return true, nil
}

func (r *FoodRepository) Delete(ctx context.Context, id string) (bool, error) {
	if _, ok := r.foods[id]; !ok {
		return false, nil
	}

	delete(r.foods, id)
	return true, nil
}

func (r *FoodRepository) Get(ctx context.Context, id string) (Food, error) {
	food, ok := r.foods[id]
	if !ok {
		return Food{}, ErrEntityNotFound
	}

	return food, nil
}

func (r *FoodRepository) List(ctx context.Context, filter FoodFilter, limit, offset int) ([]Food, int, error) {
	var foods []Food
	for _, food := range r.foods {
		if filter.Name != "" && food.Name != filter.Name {
			continue
		}

		foods = append(foods, food)

		if len(foods) == limit {
			break
		}
	}

	return foods, len(r.foods), nil
}

type RecipeRepository struct {
	recipes map[string]Recipe
}

var _ Repository[Recipe, RecipeFilter] = (*RecipeRepository)(nil)

func NewRecipeRepository() *RecipeRepository {
	return &RecipeRepository{
		recipes: make(map[string]Recipe),
	}
}

func (r *RecipeRepository) Create(ctx context.Context, recipe Recipe) (string, error) {
	recipe.ID = uuid.New().String()
	r.recipes[recipe.ID] = recipe
	return recipe.ID, nil
}

func (r *RecipeRepository) Update(ctx context.Context, recipe Recipe) (bool, error) {
	if _, ok := r.recipes[recipe.ID]; !ok {
		return false, nil
	}

	r.recipes[recipe.ID] = recipe
	return true, nil
}

func (r *RecipeRepository) Delete(ctx context.Context, id string) (bool, error) {
	if _, ok := r.recipes[id]; !ok {
		return false, nil
	}

	delete(r.recipes, id)
	return true, nil
}

func (r *RecipeRepository) Get(ctx context.Context, id string) (Recipe, error) {
	recipe, ok := r.recipes[id]
	if !ok {
		return Recipe{}, ErrEntityNotFound
	}

	return recipe, nil
}

func (r *RecipeRepository) List(ctx context.Context, filter RecipeFilter, limit, offset int) ([]Recipe, int, error) {
	var recipes []Recipe
	for _, recipe := range r.recipes {
		if filter.Name != "" && recipe.Name != filter.Name {
			continue
		}

		recipes = append(recipes, recipe)

		if len(recipes) == limit {
			break
		}
	}

	return recipes, len(r.recipes), nil
}

type (
	Food struct {
		ID       string  `json:"id"`
		Name     string  `json:"name"`
		Calories float64 `json:"calories"` // calories per 100g.
	}

	FoodFilter struct {
		Name string `json:"name"`
	}
)

type NutritionService struct {
	recipes Repository[Recipe, RecipeFilter]
	foods   Repository[Food, FoodFilter]
}

func NewNutritionService(foods Repository[Food, FoodFilter], recipes Repository[Recipe, RecipeFilter]) *NutritionService { // (foods *FoodRepository, recipes *RecipeRepository) *NutritionService {
	return &NutritionService{
		foods:   foods,
		recipes: recipes,
	}
}

var ErrInvalidRecipe = errors.New("invalid recipe")

func (s *NutritionService) CustomRecipeJob(ctx context.Context, recipe Recipe) error {
	return ErrInvalidRecipe
}

func (s *NutritionService) CreateFood(ctx context.Context, food Food) (string, error) {
	return s.foods.Create(ctx, food)
}

func (s *NutritionService) GetFood(ctx context.Context, id string) (Food, error) {
	return s.foods.Get(ctx, id)
}

func (s *NutritionService) CreateRecipe(ctx context.Context, recipe Recipe) (string, error) {
	return s.recipes.Create(ctx, recipe)
}

func (s *NutritionService) UpdateRecipe(ctx context.Context, id string, recipe Recipe) (bool, error) {
	recipe.ID = id
	return s.recipes.Update(ctx, recipe)
}

func (s *NutritionService) DeleteRecipe(ctx context.Context, id string) (bool, error) {
	return s.recipes.Delete(ctx, id)
}

var ErrEntityNotFound = errors.New("entity not found")

func (s *NutritionService) GetRecipe(ctx context.Context, id string) (Recipe, error) {
	return s.recipes.Get(ctx, id)
}

func (s *NutritionService) ListRecipes(ctx context.Context, filter RecipeFilter, limit, offset int) ([]Recipe, int, error) {
	return s.recipes.List(ctx, filter, limit, offset)
}

type CustomControllerAsService struct {
	nutrition *NutritionService // this field should be binded through the constructor. And so the rest of them, if it's a function then its returnings struct, struct's fields are not binded automatically.
	extra     *extraService
}

func NewCustomControllerAsService(nutrition *NutritionService, extra *extraService /* here more dependencies can be added */) *CustomControllerAsService {
	return &CustomControllerAsService{
		nutrition: nutrition,
		extra:     extra,
	}
}

func (c *CustomControllerAsService) Routing(r iris.Router) {
	if c.nutrition == nil {
		panic("BUG: dependency injection on constructor failed for CustomControllerAsService.nutrition")
	}

	if c.extra == nil {
		panic("BUG: dependency injection on struct fields through singleton controller failed for CustoControllerAsService.Extra (exported field)")
	}

	r.Post("/custom2", func(ctx iris.Context) {
		req, ok := ctx.BindJSON[iris.Map]()
		if !ok {
			return
		}
		ctx.JSON(iris.Map{"message": fmt.Sprintf("hello %s", req["name"])})
	})
}

type extraService struct{}

func (s *extraService) Print() string {
	return "hello!"
}

type CustomController struct {
	nutrition *NutritionService
	extra     *extraService
}

func NewCustomController(nutrition *NutritionService /* here more dependencies can be added */, extra *extraService) *CustomController {
	return &CustomController{
		nutrition: nutrition,
		extra:     extra,
	}
}

func (c *CustomController) Routing(r iris.Router) {
	r.Post("/custom3", handler.JobHandler(c.CustomRecipeJob))
	r.Get("/", func(ctx iris.Context) {
		ctx.WriteString(c.extra.Print())
	})
}

func (c *CustomController) CustomRecipeJob(ctx context.Context, recipe Recipe) error {
	return c.nutrition.CustomRecipeJob(ctx, recipe)
}

// type CustomControllerAsDepFunc func(nutrition *NutritionService) *CustomControllerAsService

func newAppNoGuide() *iris.Application {
	app := iris.New()
	opts := errors.NewOptions()
	opts.MapErrors(errors.NotFound, ErrEntityNotFound) // or just return errors.NotFound when EntityNotFound is returned instead.
	opts.MapErrors(errors.InvalidArgument, ErrInvalidRecipe)
	app.Use(opts.Handler())

	app.DI().RegisterDependency(
		NewFoodRepository,
		NewRecipeRepository,
		NewNutritionService,
	)

	app.Router("/api/foods", func(s *NutritionService) iris.Controller {
		return iris.Join(
			handler.Create(s.CreateFood),
			handler.Get(s.GetFood),
		)
	})
	// To register dependencies only for specific controller(s) see the iris.Dependencies usage below:
	// app.Controller("/api/recipes",iris.Dependencies(NewFoodRepository, NewRecipeRepository, NewNutritionService), ctrl)

	// app.Controller("/api/recipes", func(s *NutritionService) router.Controller {
	// 	return iris.Join(
	// 		handler.CreateID(s.CreateRecipe),
	// 		handler.Delete(s.DeleteDelete),
	// 		iris.Routing(func(r iris.Router) {
	// 			r.Post("/custom", s.CustomRecipeJob)
	// 			r.Put("/{recipe_id}", s.XXX)
	// 			r.Get("/{recipe_id}", s.XXX)
	// 			r.Post("/list", s.XXX)
	// 		}),
	// 	)
	// })
	// OR:
	app.Router("/api/recipes", func(s *NutritionService) iris.Controller {
		return iris.Join(
			// A custom route example, not a CRUD operation.
			iris.Prefix("/custom", handler.Job(s.CustomRecipeJob)),
			//
			handler.Create(s.CreateRecipe),
			handler.UpdateBy(s.UpdateRecipe),
			handler.Delete(s.DeleteRecipe),
			handler.Get(s.GetRecipe),
			handler.List(s.ListRecipes),
		)
	})

	return app
}

func newApp() *iris.Application {
	return iris.NewBuilder().
		Prefix("/api").
		AllowOrigin("*").
		Compression(true).
		LogRequests(false).
		Health(true, "testing", "kataras").
		// Timeout(iris.Timeout().WithServerTimeout(30*time.Second)).
		Errors(errors.NewOptions().
			MapErrors(errors.NotFound, ErrEntityNotFound).
			MapErrors(errors.InvalidArgument, ErrInvalidRecipe)).
		Services(NewFoodRepository, NewRecipeRepository, NewNutritionService, new(extraService) /* this extraService is only used on CustomerControllerAsService.Extra and NewCustomController */, NewCustomControllerAsService).
		Middlewares().
		API("/foods", func(s *NutritionService) iris.Controller {
			return iris.Join(
				handler.Create(s.CreateFood),
				handler.Get(s.GetFood),
			)
		}).
		API("/recipes", func(s *NutritionService) iris.Controller {
			return iris.Join(
				// A custom route example, not a CRUD operation.
				iris.Prefix("/custom", handler.Job(s.CustomRecipeJob)),
				//
				handler.Create(s.CreateRecipe),
				handler.UpdateBy(s.UpdateRecipe),
				handler.Delete(s.DeleteRecipe),
				handler.Get(s.GetRecipe),
				handler.List(s.ListRecipes),
			)
		}).
		// When a Controller was registered as a result of a dependency/service, then we can register it as a controller too to a specified router (API) through the ResolveController method.
		// That controller can be registered as a dependency for other controllers too.
		API("/ctrl", iris.ResolveController[CustomControllerAsService]()).
		// When you want to register a controller that has a constructor that accepts dependencies and returns a compatible controller and you don't want to register it as a service,
		// you can just pass it as a controller.
		// That controller can NOT be registered as a dependency for other controllers.
		API("/ctrl2", NewCustomController). // this now completes the need of newController too.
		API("/ctrl4", func(s *NutritionService) func(iris.Router) {
			return func(r iris.Router) {
				if s == nil {
					panic("BUG: dependency injection failed for /ctrl4")
				}
				r.Get("/", func(ctx iris.Context) {
					ctx.WriteString("OK")
				})
			}
		}).
		Build()
}

func TestController(t *testing.T) {
	app := newApp()

	testRouting(t, app, "/api")
}

func testRouting(t *testing.T, app *iris.Application, relativePath string) *httptest.Expect {
	e := httptest.New(t, app)
	doRequests(e, relativePath)
	return e
}

func doRequests(e *httptest.Expect, relativePath string) {
	// custom controller as service and controller.
	e.POST(relativePath+"/ctrl/custom2").WithJSON(iris.Map{"name": "makis"}).Expect().Status(httptest.StatusOK).JSON().Object().HasValue("message", "hello makis")
	// custom controller with function that accepts dependencies and returns a compatible controller.
	e.POST(relativePath+"/ctrl2/custom3").WithJSON(Recipe{
		Name: "test",
	}).Expect().Status(httptest.StatusBadRequest).JSON().Object().HasValue("message", "invalid recipe")
	e.GET(relativePath + "/ctrl2").Expect().Status(httptest.StatusOK).Body().IsEqual("hello!")
	// Test just a func(iris.Router)
	e.GET(relativePath + "/ctrl4").Expect().Status(httptest.StatusOK).Body().IsEqual("OK")

	// Test request to the POST /books/custom custom route.
	e.POST(relativePath+"/recipes/custom").WithJSON(Recipe{
		Name: "test",
	}).Expect().Status(httptest.StatusBadRequest).JSON().Object().HasValue("message", "invalid recipe")

	doNutritionRequests(e, relativePath)
}

func doNutritionRequests(e *httptest.Expect, relativePath string) {
	var (
		food1ID = uuid.New().String()
		food2ID = uuid.New().String()
		food3ID = uuid.New().String()
	)

	// Test request to the POST "/books" route.
	recipe1 := Recipe{
		Name: "recipe1",
		Items: []RecipeItem{
			{
				FoodID:   food1ID,
				Quantity: 255,
			},
			{
				FoodID:   food2ID,
				Quantity: 155,
			},
		},
	}
	jsonObject := e.POST(relativePath + "/recipes").WithJSON(recipe1).Expect().Status(httptest.StatusCreated).JSON().Object()
	recipe1.ID = jsonObject.Value("id").String().NotEmpty().Raw()
	// as we don't return full object for now: jsonObject.IsEqual(recipe1)

	// Test request to the GET "/books/{id}" route.
	e.GET(relativePath + "/recipes/" + recipe1.ID).Expect().Status(httptest.StatusOK).JSON().Object().IsEqual(recipe1)
	// Test request to the PUT "/books/{id}" route.
	recipe1.Name = "recipe1_updated"
	recipe1.Items = append(recipe1.Items, RecipeItem{
		FoodID:   food3ID,
		Quantity: 100,
	})
	e.PUT(relativePath + "/recipes/" + recipe1.ID).WithJSON(recipe1).Expect().Status(httptest.StatusNoContent).Body().IsEmpty()
	// Test request to the POST "/books/list" route.
	e.POST(relativePath + "/recipes/list").WithJSON(RecipeFilter{
		Name: "recipe1_updated",
	}).Expect().Status(httptest.StatusOK).JSON().IsEqual(irisContext.NewPage(
		iris.PageOptions{
			Page: 1,
			Size: 50,
		},
		RecipeFilter{
			Name: "recipe1_updated",
		},
		[]Recipe{
			recipe1,
		},
		1,
	),
	)
	// Test request to the DELETE "/books/{id}" route.
	e.DELETE(relativePath + "/recipes/" + recipe1.ID).Expect().Status(httptest.StatusNoContent).Body().IsEmpty()
	// Try to delete it again: the function returns false this time (nothing was
	// there to delete), which is a 304 Not Modified with an empty body.
	e.DELETE(relativePath + "/recipes/" + recipe1.ID).Expect().Status(httptest.StatusNotModified).Body().IsEmpty()
	// Try to get the book by its ID after DELETE, should return 404.
	e.GET(relativePath + "/recipes/" + recipe1.ID).Expect().Status(httptest.StatusNotFound)
}
