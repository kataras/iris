package apigraph

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
)

var (
	themesMu sync.RWMutex
	themes   = make(map[string]Theme)
)

func init() {
	RegisterTheme(Light())
	RegisterTheme(Dark())
}

// RegisterTheme registers a theme .
// If a theme with the same name already exists, it is overwritten.
func RegisterTheme(theme Theme) {
	if theme == nil {
		return
	}
	themesMu.Lock()
	themes[theme.GetName()] = theme
	themesMu.Unlock()
}

// lookupTheme returns the theme with the given name.
// If the theme does not exist, it returns the light theme.
func lookupTheme(name string) Theme {
	themesMu.RLock()
	theme, ok := themes[name]
	themesMu.RUnlock()
	if ok {
		return theme
	}
	return Light()
}

// Theme represents a theme for the graph.
type Theme interface {
	// GetName returns the name of the theme.
	GetName() string
	// GetStyle generates the CSS for the theme.
	GetStyle() string
}

// DefaultTheme defines the default theme structure.
type DefaultTheme struct {
	// Name is the name of the theme.
	Name string
	// BackgroundColor is the background color of the graph.
	BackgroundColor string
	// Color is the text color of the graph.
	Color string
	// ReactColor is the color used for interactive elements.
	ReactColor string
	// StrokeColor is the color used for strokes around elements.
	StrokeColor string
}

// Light returns a light theme.
// It's the default theme.
func Light() *DefaultTheme {
	return &DefaultTheme{
		Name:            "light",
		BackgroundColor: "#fff",
		Color:           "#000",
		ReactColor:      "#fff",
		StrokeColor:     "#000",
	}
}

// Dark returns a dark theme.
func Dark() *DefaultTheme {
	return &DefaultTheme{
		Name:            "dark",
		BackgroundColor: "#333",
		Color:           "#ddd",
		ReactColor:      "#444",
		StrokeColor:     "#888",
	}
}

// GetName returns the name of the theme.
func (t *DefaultTheme) GetName() string {
	return t.Name
}

// GetStyle generates the CSS for the theme.
func (t *DefaultTheme) GetStyle() string {
	return fmt.Sprintf(`
        body {
            background-color: %s;
            color: %s;
        }
        .link:hover {
            stroke: %s;
            stroke-width: 4px;
        }
        .node text {
            font: 12px sans-serif;
            fill: %s;
        }
        .node rect {
            fill: %s;
            stroke: %s;
            stroke-width: 0px;
            rx:5;
            ry:5;
            height: 20px;
            min-width: 45px;
        }
        .node a {
            text-decoration: none;
            color: inherit;
        }
    `, t.BackgroundColor, t.Color, t.ReactColor, t.Color, t.ReactColor, t.StrokeColor)
}

// Options defines the options for the Controller.
type Options struct {
	// Theme is the theme name to be used for the graph.
	// See RegisterTheme for more information.
	Theme string
}

// controller represents the API graph router.
type controller struct {
	options Options
}

var _ router.Controller = (*controller)(nil)

// New creates a new API Graph router.Controller with the given options.
// It registers a handler that renders a graph of the application's routes.
// It generates an HTML page with a D3.js visualization of the routes, showing
// the paths, methods, and handlers. The graph is interactive, allowing users
// to directly open the handler files in Visual Studio Code. Users can also
// to hover over links and nodes to see more details.
//
// Parameters:
//   - options: Options for the graph, such as the theme. If no options are provided,
//     the default theme is used. The theme determines the background color, text color,
//     and link color of the graph. The default theme is light, but a dark theme is also
//     available. To use the dark theme, pass "dark" as the theme.
//
// The generated HTML includes a D3.js script that creates a horizontal tree
// layout of the routes. Each node represents a route, and each link represents
// the relationship between parent and child routes. The nodes display the route
// paths and methods, and clicking on a node opens the corresponding handler
// file in Visual Studio Code using the vscode://file/ URL scheme.
//
// Usage:
//
//	func main() {
//	    app := iris.New()
//	    app.Router("/apigraph", apigraph.New(apigraph.Options{Theme: "dark"}))
//
//	    // Example routes with the same path but different methods:
//	    app.Get("/", func(ctx iris.Context) { ctx.WriteString("GET /") })
//	    app.Get("/users", func(ctx iris.Context) { ctx.WriteString("GET /users") })
//	    app.Get("/users/list", func(ctx iris.Context) { ctx.WriteString("GET /users/list") })
//	    app.Post("/users", func(ctx iris.Context) { ctx.WriteString("POST /users") })
//	    app.Get("/users/{id:int}", func(ctx iris.Context) { ctx.WriteString("GET /users/{id:int}") })
//	    app.Put("/users/{id:int}", func(ctx iris.Context) { ctx.WriteString("PUT /users/{id:int}") })
//	    app.Get("/users/{id:int}/posts", func(ctx iris.Context) { ctx.WriteString("GET /users/{id:int}/posts") })
//
//	    app.Listen(":8080")
//	}
func New(options Options) router.Controller {
	return &controller{
		options: options,
	}
}

// Routing registers the routes for the Controller.
func (a *controller) Routing(r router.Router) {
	if r.Application() == nil {
		panic("router is not attached to an application")
	}

	var graphsHTML []byte

	afterBuild := func() error {
		graphsData := string(getGraphsData(r.Application()))
		graphsHTML = fmt.Appendf(nil, `<!DOCTYPE html> <html> <head> <meta charset="utf-8"> <meta name="description" content="Iris Web Framework | API Graph"> <meta name="keywords" content="Go, Iris, Golang, Web Framework"> <meta name="author" content="Gerasimos (Makis) Maropoulos"> <script src="https://d3js.org/d3.v7.min.js"></script> <style> %s </style> </head> <body> <div id="graph"></div> <script> let data = %s; let width = 1024, height = 768; let svg = d3.select("#graph").append("svg") .attr("width", width).attr("height", height); let root = d3.hierarchy(data, d => Object.values(d.children || {})); /* Calculate the number of nodes */ let nodes = root.descendants(); let nodeCount = nodes.length; /* Adjust the SVG size based on the number of nodes */ let adjustedWidth = Math.max(width, nodeCount * 10.5); let adjustedHeight = Math.max(height, nodeCount * 21); svg.attr("height", adjustedHeight); svg.attr("width", adjustedWidth); /* Make tree horizontal (left -> right). */ let treeLayout = d3.tree().size([adjustedHeight - 100, adjustedWidth - 400]); treeLayout(root); let g = svg.append("g").attr("transform","translate(50,50)"); /* Links */ g.selectAll(".link") .data(root.links()) .enter().append("path") .attr("class","link") .attr("d", d3.linkHorizontal() .x(d => d.y) .y(d => d.x)) .attr("stroke","#999") .attr("fill","none"); /* Nodes */ let node = g.selectAll(".node") .data(nodes) .enter().append("g") .attr("class", "node") .attr("transform", d => "translate(" + d.y + "," + d.x + ")"); node.append("circle").attr("r", 5).attr("fill", "steelblue"); node.append("title") .text(d => { let label = d.data.fullpath || ""; if (d.data.routes) { d.data.routes.forEach(r => { label += "\n[" + r.method + "] " + r.handlers.join(", "); }); } return label; }); node.append("rect") .attr("x", 10) .attr("y", -10) .attr("width", d => { let label = d.data.name || ""; if (d.data.routes) { d.data.routes.forEach(r => { label += " [" + r.method + "]"; }); } if (label.length <= 1) { return 0; } if (label.length <= 11) { return label.length * 10; } if (label.length <= 18) { return label.length * 8; } return label.length * 6.2; /* Increase the width multiplier for more space */ }) .attr("height", 20); /* Increase the height for better readability */ node.append("text") .attr("dy", "0.35em") .attr("x", 20) /* Increase the x position for more space */ .html(d => { let label = d.data.name || ""; if (d.data.routes) { d.data.routes.forEach(r => { let color; switch (r.method) { case 'GET': color = '#28a745'; /* Greenish */ break; case 'POST': color = '#007bff'; /* Blueish */ break; case 'PUT': color = '#ffc107'; /* Yellowish */ break; case 'DELETE': color = '#dc3545'; /* Reddish */ break; case 'PATCH': color = '#17a2b8'; /* Tealish */ break; case 'OPTIONS': color = '#6c757d'; /* Grayish */ break; case 'HEAD': color = '#6610f2'; /* Purple */ break; case 'CONNECT': color = '#e83e8c'; /* Pink */ break; case 'TRACE': color = '#fd7e14'; /* Orange */ break; default: color = '#000'; /* Default to black. NOTE: colors do NOT work. */ } label += ' <a href="vscode://file/' + r.file_line + '" style="color:' + color + '">[' + r.method + ']</a>'; }); } return label; }); </script> </body> </html>`,
			// Theme colors
			lookupTheme(a.options.Theme).GetStyle(),
			// Graphs data.
			graphsData)
		return nil
	}

	r.Application().RegisterOnBuild(nil, afterBuild)

	var (
		modifiedAt = time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
		cType      = "text/html; charset=utf-8"
	)
	handler := func(ctx *context.Context) {
		ctx.SetHeader("Last-Modified", modifiedAt)
		ctx.SetHeader("Content-Type", cType)
		ctx.Write(graphsHTML)
	}

	r.Get("/", handler)
}

// routeInfo represents information about a route.
type routeInfo struct {
	// Method is the HTTP method of the route.
	Method string `json:"method"`
	// MainHandlerName is the name of the main handler for the route.
	MainHandlerName string `json:"main_handler_name"`
	// FileLine is the file and line number where the route is defined.
	FileLine string `json:"file_line"`
	// Handlers is the list of handlers for the route.
	Handlers []string `json:"handlers"`
}

// routeNode represents a node in the route tree.
type routeNode struct {
	// Name is the name of the node.
	Name string `json:"name"`
	// Routes is the list of routes for the node.
	Routes []routeInfo `json:"routes,omitempty"`
	// Children is the map of child nodes.
	Children map[string]*routeNode `json:"children,omitempty"`
	// Fullpath is the full path of the node.
	Fullpath string `json:"fullpath,omitempty"`
}

// newNode creates a new route node.
func newNode(fullpath, name string) *routeNode {
	return &routeNode{
		Name:     name,
		Children: make(map[string]*routeNode),
		Fullpath: fullpath,
	}
}

// insertPath inserts a path into the route tree.
func insertPath(root *routeNode, fileline string, fullpath string, pathParts []string, method string, handlers []string) {
	current := root
	for _, part := range pathParts {
		if part == "" {
			continue
		}
		if _, ok := current.Children[part]; !ok {
			current.Children[part] = newNode(fullpath, part)
		}
		current = current.Children[part]
	}

	mainHandlerName := ""
	if len(handlers) > 0 {
		mainHandlerName = handlers[len(handlers)-1]
	}
	// Append the method/handlers info here:
	current.Routes = append(current.Routes, routeInfo{
		Method:          method,
		Handlers:        handlers,
		MainHandlerName: mainHandlerName,
		FileLine:        fileline,
	})
}

// getGraphsData generates the graph data for the application.
func getGraphsData(app context.Application) []byte {
	root := newNode("/", "/")

	for _, r := range app.ListRoutesReadOnly() {
		parts := strings.Split(r.GetPath(), "/")
		var hNames []string
		for _, h := range r.GetHandlerList().List() {
			if context.IgnoreHandlerName(h.Name) {
				continue
			}

			hNames = append(hNames, h.Name)
		}

		fileline := fmt.Sprintf("%s:%d", r.GetRegisterFileName(), r.GetRegisterLineNumber())
		insertPath(root, fileline, r.GetPath(), parts, r.GetMethod(), hNames)
	}
	b, _ := json.Marshal(root)
	return b
}
