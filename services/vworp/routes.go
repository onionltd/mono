package main

func (s *server) routes() {
	s.router.GET("/", s.handleHome())
	s.router.GET("/health", s.handleHealthCheck())

	s.router.POST("/links/new", s.handleLinksNew())
	s.router.GET("/links/oops/:id", s.handleLinksOops(s.oopsSet["/links/oops/:id"]))
	s.router.GET("/links/:fp", s.handleLinksView())
	s.router.GET("/links/:fp/oops/:id", s.handleLinksOops(s.oopsSet["/links/:fp/oops/:id"]))

	s.router.GET("/to/*", s.handleRedirectPath())
	s.router.GET("/to/oops/:id", s.handleLinksOops(s.oopsSet["/to/oops/:id"]))
	s.router.GET("/t/:fp", s.handleRedirectFingerprint())
	s.router.GET("/t/oops/:id", s.handleLinksOops(s.oopsSet["/t/oops/:id"]))

	s.router.Static("/static", s.config.WWWDir)
}
