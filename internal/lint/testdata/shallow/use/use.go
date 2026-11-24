package use

import "fixture/tiny"

func Use(t *tiny.T) int { return t.A() + t.B() }
