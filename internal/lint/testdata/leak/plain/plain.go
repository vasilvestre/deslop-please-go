package plain

type Config struct{ A, B int }

func (c Config) Sum() int { return c.A + c.B }
