package main

func (s *server) routes() {
	s.router.File("/robots.txt", s.config.WWWDir+"/robots.txt")
	s.router.Static("/", s.config.WWWDir)
}
