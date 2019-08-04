package main

import (
	"github.com/labstack/echo"
	"github.com/zedjones/redirectprotect/routes"
)

func main() {
	e := echo.New()
	e.POST("/add_redirect", routes.RegisterURL)
	e.GET("*", routes.GetRedirect)
	e.Start(":1234")
}
