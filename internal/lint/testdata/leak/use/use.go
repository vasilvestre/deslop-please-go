package use

import "fixture/plain"

func Use(c *plain.Config) {
	c.A = 1
	c.B++
}
