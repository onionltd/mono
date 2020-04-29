package main

import serverutils "github.com/onionltd/mono/pkg/utils/echo/server"

func (s *server) routes() {
	s.router.GET("/", s.handlePage("home"))
	s.router.GET("/about", s.handlePage("about"))
	s.router.GET("/privacy", s.handlePage("privacy"))
	s.router.GET("/health", serverutils.HandleHealthCheck())

	s.router.POST("/links/new", s.handleLinksNew(), s.solveCaptcha())
	s.router.GET("/links/oops/:id", s.handleLinksOops(s.oopsSet["/links/oops/:id"], true))
	s.router.GET("/links/:fp", s.handleLinksView())
	s.router.GET("/links/:fp/oops/:id", s.handleLinksOops(s.oopsSet["/links/:fp/oops/:id"], false))

	s.router.GET("/to/:id/:fp", s.handleRedirect())
	s.router.GET("/to/oops/:id", s.handleLinksOops(s.oopsSet["/to/oops/:id"], false))

	s.router.GET("/sorry", s.handleCaptcha())

	s.router.File("/robots.txt", s.config.WWWDir+"/robots.txt")

	s.router.Static("/static", s.config.WWWDir)
}
