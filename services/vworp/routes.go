package main

func (s *server) routes() {
	s.router.GET("/", s.handleHome())
	s.router.GET("/health", s.handleHealthCheck())

	s.router.GET("/new", s.handleRedirectHome())
	s.router.POST("/new", s.handleServiceNewURL())
	s.router.GET("/to/*", s.handleServiceRedirect())

	s.router.Static("/static", s.config.WWWDir)
}
